// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
package smart

import (
	"encoding/json"
	"os"
	"strings"

	"entware-manager/internal/auth"
	"entware-manager/internal/cgiutil"
)

// smartConfigFile — маппинг устройство -> тип smartctl (-d).
// Пример: {"devices": {"sda": "sntjmicron"}}.
// Пусто или "auto" = автовыбор (nvme/sat + автоперебор USB-мостов).
var smartConfigFile = "/opt/web_entware/smart_config.json"

// allowedDeviceTypes — типы -d из smartmontools, допустимые в конфиге.
// sat,12 — вариант SAT с 12-байтными CDB для капризных USB-мостов.
var allowedDeviceTypes = []string{
	"auto",
	"sat", "sat,12",
	"scsi", "nvme", "ata",
	"usbjmicron", "usbcypress", "usbsunplus",
	"sntjmicron", "sntasmedia", "sntrealtek",
	"jmb39x-q", "jms56x",
}

var allowedDeviceTypeSet = map[string]bool{
	"auto": true,
	"sat":  true, "sat,12": true,
	"scsi": true, "nvme": true, "ata": true,
	"usbjmicron": true, "usbcypress": true, "usbsunplus": true,
	"sntjmicron": true, "sntasmedia": true, "sntrealtek": true,
	"jmb39x-q": true, "jms56x": true,
}

// usbProbeChain — порядок автоперебора для USB/SATA-дисков, когда стартовый
// тип не подошёл (мост не отвечает: unknown usb bridge / unsupported scsi
// opcode / Read Device Identity failed). nvme сюда не входит.
var usbProbeChain = []string{"sat", "usbjmicron", "sntjmicron", "scsi"}

// isValidDeviceType проверяет тип из конфига/запроса.
func isValidDeviceType(t string) bool {
	return allowedDeviceTypeSet[t]
}

// loadSmartConfig читает маппинг устройство -> тип. Ошибки/битый JSON = пусто.
func loadSmartConfig() map[string]string {
	data, err := os.ReadFile(smartConfigFile)
	if err != nil {
		return map[string]string{}
	}
	var raw struct {
		Devices map[string]string `json:"devices"`
	}
	if err := json.Unmarshal(data, &raw); err != nil || raw.Devices == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(raw.Devices))
	for dev, typ := range raw.Devices {
		dev = strings.TrimPrefix(dev, "/dev/")
		if !deviceRe.MatchString(dev) || len(dev) > 32 {
			continue
		}
		if !isValidDeviceType(typ) || typ == "auto" {
			continue
		}
		out[dev] = typ
	}
	return out
}

// configuredType возвращает ручной тип для устройства ("" = авто).
func configuredType(device string) string {
	return loadSmartConfig()[device]
}

// resolveType — единый выбор -d типа: конфиг > nvme > sat.
func resolveType(device string) string {
	if t := configuredType(device); t != "" {
		return t
	}
	if strings.HasPrefix(device, "nvme") {
		return "nvme"
	}
	return "sat"
}

// candidatesFor строит цепочку проб: ручной тип первым, затем автоперебор.
// nvme: только nvme. sat,12: сначала sat,12, затем общая цепочка без дубля sat.
func candidatesFor(device string) []string {
	if strings.HasPrefix(device, "nvme") && configuredType(device) == "" {
		return []string{"nvme"}
	}
	seen := map[string]bool{}
	var out []string
	if t := resolveType(device); t != "" {
		out = append(out, t)
		seen[t] = true
	}
	for _, t := range usbProbeChain {
		if !seen[t] {
			out = append(out, t)
			seen[t] = true
		}
	}
	return out
}

// isSmartOutputUsable — вывод smartctl пригоден (мост ответил, данные есть).
// Маркеры неуспеха: мост неизвестен, opcode не поддержан, identity не прочитана.
func isSmartOutputUsable(output string) bool {
	if output == "" {
		return false
	}
	l := strings.ToLower(output)
	for _, m := range []string{
		"unknown usb bridge",
		"unsupported scsi opcode",
		"read device identity failed",
		"mandatory smart command failed",
		"device lacks smart",
		"unable to detect device type",
	} {
		if strings.Contains(l, m) {
			return false
		}
	}
	return true
}

// HandleSmartConfig — GET отдаёт {devices, allowed}, POST сохраняет один диск.
// POST только JSON: {"device":"sda","dev_type":"sntjmicron"} или "auto" для сброса.
func HandleSmartConfig() {
	if cgiutil.IsGET() {
		cgiutil.WriteJSON(map[string]any{
			"devices": loadSmartConfigRaw(),
			"allowed": allowedDeviceTypes,
		})
		return
	}
	if !cgiutil.IsPOST() {
		cgiutil.NotAllowed()
		return
	}
	if auth.IsCrossSiteOrigin() {
		cgiutil.WriteStatusError(auth.CrossSiteDeny)
		return
	}
	body := cgiutil.ReadPOSTBody()
	if body == "" {
		cgiutil.WriteStatusError("device and dev_type required")
		return
	}
	var req struct {
		Device  string `json:"device"`
		DevType string `json:"dev_type"`
	}
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		cgiutil.WriteStatusError("invalid JSON")
		return
	}
	req.Device = strings.TrimPrefix(strings.TrimSpace(req.Device), "/dev/")
	req.DevType = strings.TrimSpace(req.DevType)
	if req.Device == "" || !deviceRe.MatchString(req.Device) || len(req.Device) > 32 {
		cgiutil.WriteStatusError("invalid device")
		return
	}
	if req.DevType == "" || req.DevType == "auto" {
		// Сброс в авто: удаляем запись.
		devices := loadSmartConfigRaw()
		delete(devices, req.Device)
		if err := saveSmartConfigRaw(devices); err != nil {
			cgiutil.WriteStatusError("Failed to write config")
			return
		}
		// Сброс кеша списка: следующий list опросит диск автовыбором.
		invalidateSmartList(req.Device)
		cgiutil.WriteJSON(map[string]string{"status": "ok", "message": "Сброшено в авто"})
		return
	}
	if !isValidDeviceType(req.DevType) {
		cgiutil.WriteStatusError("invalid dev_type")
		return
	}
	devices := loadSmartConfigRaw()
	devices[req.Device] = req.DevType
	if err := saveSmartConfigRaw(devices); err != nil {
		cgiutil.WriteStatusError("Failed to write config")
		return
	}
	// Сброс кеша списка: следующий list опросит диск с новым типом.
	invalidateSmartList(req.Device)
	cgiutil.WriteJSON(map[string]string{"status": "ok", "message": "Тип " + req.DevType + " сохранён для " + req.Device})
}

// loadSmartConfigRaw — маппинг устройство -> тип для GET/слияния.
// Раньше дублировал loadSmartConfig на 90 % (с мёртвой веткой "auto" уже
// после фильтра); теперь это копия валидированной карты — источник истины
// один, GET не может разойтись с тем, что реально применяется к дискам.
func loadSmartConfigRaw() map[string]string {
	src := loadSmartConfig()
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

// saveSmartConfigRaw атомарно пишет маппинг.
func saveSmartConfigRaw(devices map[string]string) error {
	if devices == nil {
		devices = map[string]string{}
	}
	out, err := json.MarshalIndent(map[string]any{"devices": devices}, "", "  ")
	if err != nil {
		return err
	}
	return cgiutil.WriteFileAtomic(smartConfigFile, out, 0644)
}
