"use strict";

// Подключение в шаблоне: <script src="/static/js/variations.js" defer></script>.
// POST /api/variations принимает {text} и возвращает {variations: [{text,
// description, suggested_banner}]}. HTTP-обработчик реализует команда обработчиков.
(() => {
  const MAX_TEXT_LENGTH = 1000;
  const MAX_VARIATION_LENGTH = 49;
  const MAX_DESCRIPTION_LENGTH = 29;
  // Серверу даётся 10 секунд на вызов LLM; добавляем время на передачу HTTP-запроса и ответа.
  const REQUEST_TIMEOUT_MS = 12000;
  const BANNERS = new Set(["standard", "shadow", "thinkertoy"]);
  const INVALID_RESPONSE = "Сервер вернул некорректные идеи. Попробуйте ещё раз.";

  function validText(text) {
    return typeof text === "string" && text.trim().length > 0 &&
      text.length <= MAX_TEXT_LENGTH && !/[^\x20-\x7E\r\n]/.test(text) &&
      !/\r(?!\n)/.test(text);
  }

  function initializeVariations() {
    const button = document.querySelector("#ideas-button, #get-variations-btn");
    const panel = document.querySelector("#variations-panel, #variations-modal");
    const input = document.querySelector('[name="inputWord"]');
    const bannerSelect = document.querySelector("select#banners");
    const bannerRadios = Array.from((input?.form || document)
      .querySelectorAll('input[type="radio"][name="banners"]'));

    if (!button || !panel || !input || (!bannerSelect && !bannerRadios.length)) return;
    if (button.dataset.variationsInitialized === "true") return;
    button.dataset.variationsInitialized = "true";
    button.type = "button";
    button.setAttribute("aria-controls", panel.id);
    button.setAttribute("aria-expanded", "false");

    const title = document.createElement("h3");
    title.id = "variations-title";
    title.className = "small-label";
    title.textContent = "Идеи для текста";

    const closeButton = document.createElement("button");
    closeButton.type = "button";
    closeButton.className = "text-button";
    closeButton.textContent = "Закрыть идеи";

    const status = document.createElement("p");
    status.className = "hint";
    status.setAttribute("role", "status");
    status.setAttribute("aria-live", "polite");

    const list = document.createElement("div");
    list.className = "variations";
    panel.replaceChildren(title, closeButton, status, list);
    panel.setAttribute("role", "region");
    panel.setAttribute("aria-labelledby", title.id);
    panel.hidden = true;

    let activeRequest = null;

    function setLoading(loading) {
      button.disabled = loading;
      button.setAttribute("aria-busy", String(loading));
      panel.setAttribute("aria-busy", String(loading));
    }

    function cancelRequest() {
      const request = activeRequest;
      activeRequest = null;
      if (request) {
        clearTimeout(request.timeout);
        request.controller.abort();
      }
      setLoading(false);
    }

    function closeIdeas() {
      cancelRequest();
      panel.hidden = true;
      panel.classList.add("hidden");
      button.setAttribute("aria-expanded", "false");
      list.replaceChildren();
      status.textContent = "";
    }

    function showMessage(message) {
      panel.hidden = false;
      panel.classList.remove("hidden");
      button.setAttribute("aria-expanded", "true");
      list.replaceChildren();
      status.textContent = message;
    }

    function readVariations(data) {
      if (!data || !Array.isArray(data.variations) ||
          data.variations.length < 3 || data.variations.length > 5) {
        throw new Error(INVALID_RESPONSE);
      }
      const seenTexts = new Set();
      for (const variation of data.variations) {
        if (!variation || !validText(variation.text) ||
            variation.text.length > MAX_VARIATION_LENGTH || /[\r\n]/.test(variation.text) ||
            typeof variation.description !== "string" ||
            !variation.description.trim() ||
            Array.from(variation.description).length > MAX_DESCRIPTION_LENGTH ||
            /[\x00-\x1F\x7F-\x9F]/.test(variation.description) ||
            !BANNERS.has(variation.suggested_banner) || seenTexts.has(variation.text)) {
          throw new Error(INVALID_RESPONSE);
        }
        seenTexts.add(variation.text);
      }
      return data.variations;
    }

    function selectVariation(variation) {
      // Поддерживаем радиокнопки студии и исходный выпадающий список.
      const banners = bannerSelect ? Array.from(bannerSelect.options) : bannerRadios;
      const option = banners.find((item) =>
        item.value === variation.suggested_banner ||
        item.value === `${variation.suggested_banner}.txt`);
      if (!option) {
        showMessage("Предложенный стиль недоступен в этой форме.");
        return;
      }

      closeIdeas();
      input.value = variation.text;
      if (bannerSelect) {
        bannerSelect.value = option.value;
      } else {
        option.checked = true;
      }
      // Уведомляем подсказки, счётчики символов и остальные обработчики событий формы.
      input.dispatchEvent(new Event("input", { bubbles: true }));
      const bannerControl = bannerSelect || option;
      bannerControl.dispatchEvent(new Event("input", { bubbles: true }));
      bannerControl.dispatchEvent(new Event("change", { bubbles: true }));
      input.focus();
    }

    function renderVariations(variations) {
      const items = variations.map((variation) => {
        const choice = document.createElement("button");
        choice.type = "button";
        choice.className = "variation";
        const text = document.createElement("strong");
        text.textContent = variation.text;
        const description = document.createElement("span");
        description.textContent = `${variation.description} · ${variation.suggested_banner}`;
        choice.append(text, description);
        choice.addEventListener("click", () => selectVariation(variation));
        return choice;
      });
      list.replaceChildren(...items);
      status.textContent = "Выберите идею, чтобы использовать её текст и стиль.";
    }

    async function getVariations() {
      cancelRequest();
      const text = input.value;
      if (!validText(text)) {
        showMessage("Введите от 1 до 1000 символов: латинские буквы, цифры, знаки и переносы строк.");
        input.focus();
        return;
      }

      const request = { text, controller: new AbortController(), timedOut: false };
      activeRequest = request;
      request.timeout = setTimeout(() => {
        request.timedOut = true;
        request.controller.abort();
      }, REQUEST_TIMEOUT_MS);
      showMessage("Придумываем идеи…");
      setLoading(true);

      // Сравнение объектов запросов защищает от устаревшего ответа даже после изменения A → B → A.
      function isCurrentRequest() {
        return activeRequest === request && input.value === text;
      }

      try {
        const response = await fetch("/api/variations", {
          method: "POST",
          headers: { "Content-Type": "application/json", Accept: "application/json" },
          body: JSON.stringify({ text }),
          signal: request.controller.signal,
        });
        if (!isCurrentRequest()) return;
        if (request.timedOut) throw new Error("Request timed out");
        if (!response.ok) {
          const message = response.status === 400 ? "Проверьте введённый текст и попробуйте ещё раз." :
            response.status === 503 ? "Сервис идей недоступен. Попробуйте позже." :
            "Не удалось создать идеи. Попробуйте ещё раз.";
          throw new Error(message);
        }
        let data;
        try {
          data = await response.json();
        } catch {
          throw new Error(INVALID_RESPONSE);
        }
        if (!isCurrentRequest()) return;
        if (request.timedOut) throw new Error("Request timed out");
        renderVariations(readVariations(data));
      } catch (error) {
        if (!isCurrentRequest()) return;
        if (request.timedOut) {
          showMessage("Сервер слишком долго отвечает. Попробуйте ещё раз.");
        } else if (!request.controller.signal.aborted) {
          showMessage(error instanceof TypeError ?
            "Не удалось связаться с сервером. Проверьте соединение и попробуйте ещё раз." : error.message);
        }
      } finally {
        clearTimeout(request.timeout);
        if (activeRequest === request) {
          activeRequest = null;
          setLoading(false);
        }
      }
    }

    button.addEventListener("click", getVariations);
    closeButton.addEventListener("click", () => { closeIdeas(); button.focus(); });
    input.addEventListener("input", closeIdeas);
    input.addEventListener("change", closeIdeas);
    document.addEventListener("keydown", (event) => {
      if (event.key === "Escape" && !panel.hidden) {
        closeIdeas();
        button.focus();
      }
    });
    if (input.form) input.form.addEventListener("submit", closeIdeas);
    window.addEventListener("pagehide", cancelRequest);
    setLoading(false);
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", initializeVariations, { once: true });
  } else {
    initializeVariations();
  }
})();
