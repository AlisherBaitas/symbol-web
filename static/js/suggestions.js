"use strict";

// [Suggestions] Клиентская часть подсказок (автодополнение текста под полем ввода).
// Работает на index.html вместе с:
//   - разметкой #suggestions-dropdown в templates/index.html;
//   - стилями .suggestions / .suggestion в static/css/studio.css;
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

    if (!input || !dropdown) return;
    if (input.dataset.suggestionsInitialized === "true") return;
    input.dataset.suggestionsInitialized = "true";
    const area = document.querySelector("#suggestions-area") || dropdown;
    const counter = document.querySelector("#char-count");

    const existingStatus = document.querySelector("#suggestions-status");
    const status = existingStatus || document.createElement("p");
    status.className = "hint";
    status.setAttribute("role", "status");
    status.setAttribute("aria-live", "polite");

    const list = document.createElement("ul");
    list.className = "suggestions-list";
    if (existingStatus) dropdown.replaceChildren(list);
    else dropdown.replaceChildren(status, list);
    dropdown.setAttribute("aria-label", "Варианты продолжения");
    dropdown.hidden = true;
    area.hidden = true;
    if (counter) counter.textContent = `${input.value.length} / ${MAX_TEXT_LENGTH}`;

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
      area.hidden = true;
      dropdown.classList.add("hidden");
      list.replaceChildren();
      status.textContent = "";
    }

    function showMessage(message) {
      dropdown.hidden = false;
      area.hidden = false;
      dropdown.classList.remove("hidden");
      list.replaceChildren();
      status.textContent = message;
    }

    // [Suggestions] Проверяем ответ сервера, прежде чем показывать его пользователю.
    function readSuggestions(data) {
      if (!data || !Array.isArray(data.suggestions) || data.suggestions.length === 0 ||
          !data.suggestions.every((value) => typeof value === "string" && value.trim())) {
        throw new Error("Сервер вернул некорректные подсказки. Попробуйте ещё раз.");
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
      status.textContent = "Выберите вариант или продолжайте писать.";
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
      showMessage("Подбираем продолжения…");
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
        if (request.timedOut) throw new Error("Request timed out");
        if (!response.ok) {
          const message = response.status === 400 ? "Проверьте введённый текст и попробуйте ещё раз." :
            response.status === 503 ? "Подсказки временно недоступны. Попробуйте позже." :
            "Не удалось загрузить подсказки. Попробуйте ещё раз.";
          throw new Error(message);
        }
        let data;
        try {
          data = await response.json();
        } catch {
          throw new Error("Сервер вернул некорректные подсказки. Попробуйте ещё раз.");
        }
        if (!isCurrentRequest()) return;
        if (request.timedOut) throw new Error("Request timed out");
        renderSuggestions(readSuggestions(data));
      } catch (error) {
        if (!isCurrentRequest()) return;
        if (request.timedOut) {
          showMessage("Сервер отвечает слишком долго. Попробуйте ещё раз.");
        } else if (!request.controller.signal.aborted) {
          showMessage(error instanceof TypeError ?
            "Нет ответа от сервера. Проверьте соединение и попробуйте ещё раз." : error.message);
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
      if (counter) counter.textContent = `${input.value.length} / ${MAX_TEXT_LENGTH}`;
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
