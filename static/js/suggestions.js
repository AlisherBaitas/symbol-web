"use strict";

// [Suggestions] Клиентская часть подсказок (автодополнение текста под полем ввода).
// Работает на index.html вместе с:
//   - разметкой #suggestions-dropdown в templates/index.html;
//   - стилями .suggestions / .suggestion в static/css/style.css;
//   - обработчиком handlers.HandleSuggestion (internal/handlers/ai.go);
//   - логикой ai.GetSuggestions (internal/ai/suggestions.go).
// Подключение в шаблоне: <script src="/static/js/suggestions.js" defer></script>.
// POST /api/suggest принимает {text} и возвращает {suggestions: [строки]}.
// Подсказки запрашиваются автоматически через 300 мс после того, как пользователь перестал печатать.
(() => {
  const MIN_TEXT_LENGTH = 3;
  const MAX_TEXT_LENGTH = 1000;
  const DEBOUNCE_MS = 300;
  // Серверу даётся 10 секунд на вызов LLM; добавляем время на передачу HTTP-запроса и ответа.
  const REQUEST_TIMEOUT_MS = 12000;

  function validText(text) {
    return typeof text === "string" && text.trim().length >= MIN_TEXT_LENGTH &&
      text.length <= MAX_TEXT_LENGTH && !/[^\x20-\x7E\r\n]/.test(text);
  }

  function initializeSuggestions() {
    const input = document.querySelector('[name="inputWord"]');
    const dropdown = document.querySelector("#suggestions-dropdown");

    // На studio.html подсказки обслуживает studio.js.
    if (!input || !dropdown) return;
    if (input.dataset.suggestionsInitialized === "true") return;
    input.dataset.suggestionsInitialized = "true";

    const status = document.createElement("p");
    status.setAttribute("role", "status");
    status.setAttribute("aria-live", "polite");

    const list = document.createElement("ul");
    dropdown.replaceChildren(status, list);
    dropdown.setAttribute("aria-label", "Text suggestions");
    dropdown.hidden = true;

    let debounce = null;
    let activeRequest = null;
    // Не запрашиваем подсказки повторно для текста, который пользователь только что выбрал.
    let skipNextInput = false;

    function cancelRequest() {
      clearTimeout(debounce);
      const request = activeRequest;
      activeRequest = null;
      if (request) {
        clearTimeout(request.timeout);
        request.controller.abort();
      }
      dropdown.removeAttribute("aria-busy");
    }

    function hideSuggestions() {
      cancelRequest();
      dropdown.hidden = true;
      dropdown.classList.add("hidden");
      list.replaceChildren();
      status.textContent = "";
    }

    function showMessage(message) {
      dropdown.hidden = false;
      dropdown.classList.remove("hidden");
      list.replaceChildren();
      status.textContent = message;
    }

    // [Suggestions] Проверяем ответ сервера, прежде чем показывать его пользователю.
    function readSuggestions(data) {
      if (!data || !Array.isArray(data.suggestions) || data.suggestions.length === 0 ||
          !data.suggestions.every((value) => typeof value === "string" && value.trim())) {
        throw new Error("The server returned invalid suggestions. Please try again.");
      }
      return data.suggestions;
    }

    // [Suggestions] Клик по подсказке подставляет её текст в поле ввода.
    function selectSuggestion(value) {
      hideSuggestions();
      input.value = value;
      skipNextInput = true;
      // Уведомляем остальные обработчики формы, например variations.js.
      input.dispatchEvent(new Event("input", { bubbles: true }));
      input.focus();
    }

    // [Suggestions] Каждая подсказка — кнопка; текст вставляется через textContent (без HTML-инъекций).
    function renderSuggestions(suggestions) {
      const items = suggestions.map((value) => {
        const item = document.createElement("li");
        const choice = document.createElement("button");
        choice.type = "button";
        choice.className = "suggestion";
        choice.textContent = value;
        choice.addEventListener("click", () => selectSuggestion(value));
        item.append(choice);
        return item;
      });
      list.replaceChildren(...items);
      status.textContent = "Choose a suggestion or keep typing.";
    }

    // [Suggestions] Запрос к POST /api/suggest с тайм-аутом и защитой от устаревших ответов.
    async function getSuggestions() {
      cancelRequest();
      const text = input.value;
      if (!validText(text)) return;

      const request = { text, controller: new AbortController(), timedOut: false };
      activeRequest = request;
      request.timeout = setTimeout(() => {
        request.timedOut = true;
        request.controller.abort();
      }, REQUEST_TIMEOUT_MS);
      showMessage("Loading suggestions…");
      dropdown.setAttribute("aria-busy", "true");

      // Сравнение объектов запросов защищает от устаревшего ответа даже после изменения A → B → A.
      function isCurrentRequest() {
        return activeRequest === request && input.value === text;
      }

      try {
        const response = await fetch("/api/suggest", {
          method: "POST",
          headers: { "Content-Type": "application/json", Accept: "application/json" },
          body: JSON.stringify({ text }),
          signal: request.controller.signal,
        });
        if (!isCurrentRequest()) return;
        if (!response.ok) {
          const message = response.status === 400 ? "The text was rejected. Check your input and try again." :
            response.status === 503 ? "The suggestions service is unavailable. Please try again later." :
            "Could not load suggestions. Please try again.";
          throw new Error(message);
        }
        let data;
        try {
          data = await response.json();
        } catch {
          throw new Error("The server returned invalid suggestions. Please try again.");
        }
        if (!isCurrentRequest()) return;
        renderSuggestions(readSuggestions(data));
      } catch (error) {
        if (!isCurrentRequest()) return;
        if (request.timedOut) {
          showMessage("The server took too long to respond. Please try again.");
        } else if (!request.controller.signal.aborted) {
          showMessage(error instanceof TypeError ?
            "Could not connect to the server. Check your connection and try again." : error.message);
        }
      } finally {
        clearTimeout(request.timeout);
        if (activeRequest === request) {
          activeRequest = null;
          dropdown.removeAttribute("aria-busy");
        }
      }
    }

    // [Suggestions] Debounce: запрос уходит через 300 мс после последнего нажатия клавиши.
    input.addEventListener("input", () => {
      hideSuggestions();
      if (skipNextInput) {
        skipNextInput = false;
        return;
      }
      if (validText(input.value)) debounce = setTimeout(getSuggestions, DEBOUNCE_MS);
    });
    input.addEventListener("keydown", (event) => {
      if (event.key === "Escape") hideSuggestions();
    });
    // Клик вне поля и списка закрывает подсказки.
    document.addEventListener("click", (event) => {
      if (!dropdown.hidden && !dropdown.contains(event.target) && event.target !== input) {
        hideSuggestions();
      }
    });
    if (input.form) input.form.addEventListener("submit", hideSuggestions);
    window.addEventListener("pagehide", cancelRequest);
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", initializeSuggestions, { once: true });
  } else {
    initializeSuggestions();
  }
})();
