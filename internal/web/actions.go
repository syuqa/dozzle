package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/container"
	container_support "github.com/amir20/dozzle/internal/support/container"
	support_web "github.com/amir20/dozzle/internal/support/web"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

func (h *handler) findContainerWithActions(w http.ResponseWriter, r *http.Request) (*container_support.ContainerService, bool) {
	id := chi.URLParam(r, "id")

	userLabels := h.config.Labels
	permit := true
	if h.config.Authorization.Provider != NONE {
		user := auth.UserFromContext(r.Context())
		if user.ContainerLabels.Exists() {
			userLabels = user.ContainerLabels
		}
		permit = user.Roles.Has(auth.Actions)
	}

	if !permit {
		log.Warn().Msg("user is not permitted to perform actions on container")
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return nil, false
	}

	containerService, err := h.hostService.FindContainer(hostKey(r), id, userLabels)
	if err != nil {
		log.Error().Err(err).Msg("error while trying to find container")
		http.Error(w, err.Error(), http.StatusNotFound)
		return nil, false
	}

	return containerService, true
}

func (h *handler) containerActions(w http.ResponseWriter, r *http.Request) {
	action := chi.URLParam(r, "action")

	containerService, ok := h.findContainerWithActions(w, r)
	if !ok {
		return
	}

	parsedAction, err := container.ParseContainerAction(action)
	if err != nil {
		log.Error().Err(err).Msg("error while trying to parse action")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := containerService.Action(r.Context(), parsedAction); err != nil {
		log.Error().Err(err).Msg("error while trying to perform container action")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Info().Str("action", action).Str("container", containerService.Container.Name).Msg("container action performed")
	http.Error(w, "", http.StatusNoContent)
}

func (h *handler) containerUpdate(w http.ResponseWriter, r *http.Request) {
	containerService, ok := h.findContainerWithActions(w, r)
	if !ok {
		return
	}

	sse, err := support_web.NewSSEWriter(r.Context(), w, r)
	if err != nil {
		log.Error().Err(err).Msg("error creating SSE writer")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer sse.Close()

	progressCh := make(chan container.UpdateProgress, 50)
	errCh := make(chan error, 1)

	go func() {
		_, err := containerService.Update(r.Context(), progressCh)
		errCh <- err
	}()

	for progress := range progressCh {
		if err := sse.Event("update-progress", progress); err != nil {
			log.Error().Err(err).Msg("error writing SSE event")
			return
		}
	}

	if err := <-errCh; err != nil {
		log.Error().Err(err).Msg("container update failed")
	}

	log.Info().Str("container", containerService.Container.Name).Msg("container update completed")
}

type injectLogsButtonRequest struct {
	IndexPath string `json:"indexPath"`
	Alias     string `json:"alias"`
	LogsURL   string `json:"logsUrl"`
}

const defaultActionTimeout = 20 * time.Second

type waitForContextExecEventReader struct {
	ctx context.Context
}

func (r *waitForContextExecEventReader) Interactive() bool {
	return false
}

func (r *waitForContextExecEventReader) ReadEvent() (*container.ExecEvent, error) {
	<-r.ctx.Done()
	return nil, io.EOF
}

type closedExecEventReader struct{}

func (r *closedExecEventReader) Interactive() bool {
	return false
}

func (r *closedExecEventReader) ReadEvent() (*container.ExecEvent, error) {
	return nil, io.EOF
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

func buildInjectLogsButtonScript(logsURL string) string {
	return fmt.Sprintf(`(function () {
  var LOGS_URL = %s;

  var style = document.createElement('style');
  style.textContent = [
    '.un-logs-widget {',
    '  height: 60px;',
    '  left: auto;',
    '  margin-left: auto;',
    '  pointer-events: none;',
    '  position: fixed;',
    '  right: 0px;',
    '  top: 470px;',
    '  width: 58px;',
    '  z-index: 9999;',
    '}',
    '.un-logs-button {',
    '  pointer-events: auto;',
    '  transform: translate(54px);',
    '  transition: transform 0.3s;',
    '  display: block;',
    '  text-decoration: none;',
    '  position: relative;',
    '  height: 60px;',
    '}',
    '.un-logs-widget:hover .un-logs-button {',
    '  transform: translate(0px);',
    '}',
    '.un-logs-shadow {',
    '  box-shadow: 0 1px 2px 0 var(--un-theme-platform-wiki-wikiButton-shadowColor, rgba(0,0,0,0.25));',
    '  height: 100%%;',
    '  position: relative;',
    '}',
    '.un-logs-label {',
    '  zoom: 1.12;',
    '  background-color: var(--un-theme-platform-wiki-wikiButton-backgroundColor, #1c5c7a);',
    '  color: var(--un-theme-platform-wiki-wikiButton-textColor, #ffffff);',
    '  cursor: default;',
    '  font-size: 12px;',
    '  height: 100%%;',
    '  left: -1px;',
    '  line-height: 20px;',
    '  position: absolute;',
    '  text-align: center;',
    '  transform: rotate(180deg);',
    '  user-select: none;',
    '  writing-mode: tb-rl;',
    '}',
    '.un-logs-icon {',
    '  background-color: var(--un-theme-platform-wiki-wikiButton-backgroundColor, #1c5c7a);',
    '  cursor: pointer;',
    '  height: 60px;',
    '  margin-left: 20px;',
    '  width: 40px;',
    '  display: flex;',
    '  align-items: center;',
    '  justify-content: center;',
    '}',
    '.un-logs-icon svg {',
    '  width: 22px;',
    '  height: 22px;',
    '  display: block;',
    '  stroke: var(--un-theme-platform-wiki-wikiButton-textColor, #ffffff);',
    '}'
  ].join('\n');
  document.head.appendChild(style);

  var existing = document.querySelector('.un-logs-widget');
  if (existing) {
    existing.remove();
  }

  var widget = document.createElement('div');
  widget.className = 'un-logs-widget';
  widget.innerHTML =
    '<a class="un-logs-button" href="' + LOGS_URL + '" target="_blank" rel="noopener noreferrer" aria-label="Открыть логи">' +
      '<div class="un-logs-shadow">' +
        '<div class="un-logs-label">логи</div>' +
        '<div class="un-logs-icon">' +
          '<svg viewBox="0 0 24 24" fill="none" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">' +
            '<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>' +
            '<polyline points="14 2 14 8 20 8"/>' +
            '<line x1="16" y1="13" x2="8" y2="13"/>' +
            '<line x1="16" y1="17" x2="8" y2="17"/>' +
            '<polyline points="10 9 9 9 8 9"/>' +
          '</svg>' +
        '</div>' +
      '</div>' +
    '</a>';

  document.body.appendChild(widget);
})();`, strconv.Quote(logsURL))
}

func buildInjectLogsButtonCommand(indexPath, logsURL string) string {
	return fmt.Sprintf(`set -eu
INDEX=%s
SCRIPT="$(dirname "$INDEX")/logs-tab.js"
cat > "$SCRIPT" <<'JSEOF'
%s
JSEOF

if ! grep -q 'logs-tab.js' "$INDEX"; then
  sed -i 's|</body>|<script src="/logs-tab.js"></script></body>|' "$INDEX"
fi
`, shellQuote(indexPath), buildInjectLogsButtonScript(logsURL))
}

func (h *handler) containerInjectLogsButton(w http.ResponseWriter, r *http.Request) {
	containerService, ok := h.findContainerWithActions(w, r)
	if !ok {
		return
	}

	var payload injectLogsButtonRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	payload.IndexPath = strings.TrimSpace(payload.IndexPath)
	payload.Alias = strings.TrimSpace(payload.Alias)
	payload.LogsURL = strings.TrimSpace(payload.LogsURL)

	if payload.IndexPath == "" {
		http.Error(w, "indexPath is required", http.StatusBadRequest)
		return
	}
	if payload.Alias == "" {
		http.Error(w, "alias is required", http.StatusBadRequest)
		return
	}
	if payload.LogsURL == "" {
		http.Error(w, "logsUrl is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), defaultActionTimeout)
	defer cancel()

	err := executeInjectLogsButton(ctx, containerService, payload)
	if err != nil {
		log.Error().Err(err).Str("container", containerService.Container.Name).Msg("error while injecting logs button")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Info().Str("container", containerService.Container.Name).Str("index", payload.IndexPath).Msg("logs button injected")
	http.Error(w, "", http.StatusNoContent)
}

func executeInjectLogsButton(ctx context.Context, containerService *container_support.ContainerService, payload injectLogsButtonRequest) error {
	var stdout bytes.Buffer
	return containerService.Exec(ctx, []string{"sh", "-lc", buildInjectLogsButtonCommand(payload.IndexPath, payload.LogsURL)}, &waitForContextExecEventReader{ctx: ctx}, &stdout)
}
