package web

import (
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

var sensitiveEnvNamePatterns = []string{
	"PASSWORD",
	"PASSWD",
	"SECRET",
	"TOKEN",
	"API_KEY",
	"PRIVATE_KEY",
	"ACCESS_KEY",
	"SECRET_KEY",
	"CREDENTIAL",
	"AUTH",
}

const customEnvMaskPatternsVar = "DOZZLE_ENV_MASK_PATTERNS"

func (h *handler) getContainerEnv(w http.ResponseWriter, r *http.Request) {
	host := hostKey(r)
	containerID := chi.URLParam(r, "id")
	log.Debug().Str("host", host).Str("container", containerID).Msg("fetching container env")

	containerService, err := h.hostService.FindContainer(host, containerID, h.resolveLabels(r))
	if err != nil {
		log.Warn().Err(err).Str("host", host).Str("container", containerID).Msg("failed to fetch container env")
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	env := make(map[string]string, len(containerService.Container.Env))
	for _, item := range containerService.Container.Env {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		parts := strings.SplitN(item, "=", 2)
		key := strings.TrimSpace(parts[0])
		if key == "" || isSensitiveEnvName(key) {
			continue
		}
		value := ""
		if len(parts) == 2 {
			value = parts[1]
		}
		env[key] = value
	}

	log.Debug().Str("host", host).Str("container", containerID).Int("count", len(env)).Msg("container env fetched")
	writeJSON(w, http.StatusOK, env)
}

func isSensitiveEnvName(key string) bool {
	upper := strings.ToUpper(strings.TrimSpace(key))
	if upper == "" {
		return false
	}
	for _, pattern := range envMaskPatterns() {
		pattern = strings.ToUpper(strings.TrimSpace(pattern))
		if pattern == "" {
			continue
		}
		if strings.ContainsAny(pattern, "*?[") {
			if ok, _ := path.Match(pattern, upper); ok {
				return true
			}
			continue
		}
		if strings.Contains(upper, pattern) {
			return true
		}
	}
	return false
}

func envMaskPatterns() []string {
	custom := strings.TrimSpace(os.Getenv(customEnvMaskPatternsVar))
	if custom == "" {
		return sensitiveEnvNamePatterns
	}

	patterns := make([]string, 0, len(sensitiveEnvNamePatterns)+4)
	patterns = append(patterns, sensitiveEnvNamePatterns...)
	for _, item := range strings.Split(custom, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		patterns = append(patterns, item)
	}
	return patterns
}
