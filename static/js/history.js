"use strict";

// [Fix] Новый файл. В index.html/result.html уже были кнопка «Копировать»,
// «Попробовать с Hello, world!» и блок «Недавние надписи», но ни одного
// обработчика для них не было — кнопки ничего не делали.
// Подключение в шаблоне: <script src="/static/js/history.js" defer></script>.
// История (бонус ТЗ) хранит только текст и стиль, максимум 8 записей,
// в localStorage этого браузера; на сервер ничего не отправляется.
(() => {
  const STORAGE_KEY = "symbol-web-history";
  const MAX_ITEMS = 8;
  const EXAMPLE_TEXT = "Hello, world!";

  // Если localStorage запрещён (приватный режим), история живёт до перезагрузки.
  let memoryHistory = [];

  function loadHistory() {
    try {
      const raw = window.localStorage.getItem(STORAGE_KEY);
      const items = raw ? JSON.parse(raw) : [];
      return Array.isArray(items) ? items.filter((item) =>
        item && typeof item.text === "string" && typeof item.banner === "string") : [];
    } catch {
      return memoryHistory;
    }
  }

  function saveHistory(items) {
    memoryHistory = items;
    try {
      window.localStorage.setItem(STORAGE_KEY, JSON.stringify(items));
    } catch {
      // Хранилище недоступно — остаётся копия в памяти.
    }
  }

  function initialize() {
    const form = document.querySelector("#art-form");
    const input = document.querySelector('[name="inputWord"]');
    const output = document.querySelector("#art-output");
    const status = document.querySelector("#output-status");
    const copyButton = document.querySelector("#copy-button");
    const exampleButton = document.querySelector("#example-button");
    const historyList = document.querySelector("#history-list");
    const historyEmpty = document.querySelector("#history-empty");
    const clearButton = document.querySelector("#clear-history");

    if (!form || !input) return;

    function checkedBanner() {
      const radio = form.querySelector('input[name="banners"]:checked');
      return radio ? radio.value : "standard.txt";
    }

    function selectBanner(value) {
      const radio = form.querySelector(`input[name="banners"][value="${CSS.escape(value)}"]`);
      if (radio) radio.checked = true;
    }

    // Подставляем текст и стиль и отправляем обычную форму POST /symbol-art.
    function generate(text, banner) {
      input.value = text;
      selectBanner(banner);
      input.dispatchEvent(new Event("input", { bubbles: true }));
      if (typeof form.requestSubmit === "function") form.requestSubmit();
      else form.submit();
    }

    // --- Копирование результата ---
    if (copyButton && output) {
      copyButton.addEventListener("click", async () => {
        const art = output.textContent;
        if (!art) return;
        try {
          await navigator.clipboard.writeText(art);
          if (status) status.textContent = "Скопировано в буфер обмена.";
        } catch {
          // Буфер обмена запрещён — выделяем текст, чтобы скопировать вручную (Cmd/Ctrl+C).
          const range = document.createRange();
          range.selectNodeContents(output);
          const selection = window.getSelection();
          selection.removeAllRanges();
          selection.addRange(range);
          if (status) status.textContent = "Не удалось скопировать автоматически: текст выделен, нажмите Cmd/Ctrl+C.";
        }
      });
    }

    // --- Пример ---
    if (exampleButton) {
      exampleButton.disabled = false;
      exampleButton.addEventListener("click", () => generate(EXAMPLE_TEXT, checkedBanner()));
    }

    // --- История ---
    function renderHistory() {
      if (!historyList) return;
      const items = loadHistory();
      const buttons = items.map((item) => {
        const button = document.createElement("button");
        button.type = "button";
        button.className = "history-item";
        const label = item.text.replace(/\s+/g, " ");
        button.textContent = `${label} · ${item.banner.replace(/\.txt$/, "")}`;
        button.title = "Повторить эту надпись";
        button.addEventListener("click", () => generate(item.text, item.banner));
        return button;
      });
      historyList.replaceChildren(...buttons);
      if (clearButton) clearButton.hidden = items.length === 0;
      if (historyEmpty) historyEmpty.hidden = items.length > 0;
    }

    // Страница пришла с готовым артом -> запоминаем текст и стиль.
    if (output && !output.hidden && output.textContent && input.value.trim()) {
      const entry = { text: input.value, banner: checkedBanner() };
      const rest = loadHistory().filter((item) =>
        item.text !== entry.text || item.banner !== entry.banner);
      saveHistory([entry, ...rest].slice(0, MAX_ITEMS));
    }

    if (clearButton) {
      clearButton.addEventListener("click", () => {
        saveHistory([]);
        renderHistory();
      });
    }

    renderHistory();
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", initialize, { once: true });
  } else {
    initialize();
  }
})();
