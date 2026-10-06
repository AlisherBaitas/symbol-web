"use strict";

(() => {
  const form = document.querySelector("#art-form");
  if (!form) return;
  const $ = (selector) => document.querySelector(selector);
  const input = $("#text-input");
  const output = $("#art-output");
  const generateButton = $("#generate-button");
  const recommendButton = $("#recommend-button");
  const ideasButton = $("#ideas-button");
  const copyButton = $("#copy-button");
  const downloadButton = $("#download-button");
  const historyKey = "symbol-studio.history.v1";
  const banners = new Set(["standard", "shadow", "thinkertoy"]);
  const requests = new Map();
  let debounce;
  let history = [];
  let currentArt = output.textContent;
  let currentBanner = form.elements.banners.value;
  let storageAvailable = true;
  let assistantVersion = 0;
  let artVersion = 0;

  function validText(value) {
    return value.length > 0 && value.length <= 1000 && !/[^\x20-\x7E\n\r]/.test(value) && !/\r(?!\n)/.test(value);
  }

  function validate(showError = true) {
    const ok = validText(input.value);
    if (showError) {
      const message = !input.value ? "Напишите текст для надписи." : "До 1000 символов: латинские буквы, цифры, знаки и переносы строк.";
      $("#input-error").textContent = ok ? "" : message;
      $("#input-error").hidden = ok;
      input.setAttribute("aria-invalid", String(!ok));
    }
    return ok;
  }

  function cancel(key) {
    const request = requests.get(key);
    if (request) request.abort();
    requests.delete(key);
  }

  async function request(key, url, options) {
    cancel(key);
    const controller = new AbortController();
    requests.set(key, controller);
    let timedOut = false;
    const timeout = setTimeout(() => { timedOut = true; controller.abort(); }, 12000);
    try {
      const response = await fetch(url, { ...options, signal: controller.signal, headers: { Accept: "application/json", ...options.headers } });
      const data = await response.json();
      if (controller.signal.aborted || requests.get(key) !== controller) throw new DOMException("Запрос отменён", "AbortError");
      if (!response.ok) throw new Error(data.error || "Не удалось выполнить запрос. Попробуйте ещё раз.");
      return data;
    } catch (error) {
      if (timedOut) throw new Error("Сервер отвечает слишком долго. Попробуйте ещё раз.");
      if (error.name === "AbortError") throw error;
      if (error instanceof TypeError || error instanceof SyntaxError) throw new Error("Нет ответа от сервера. Проверьте соединение и повторите попытку.");
      throw error;
    } finally {
      clearTimeout(timeout);
      if (requests.get(key) === controller) requests.delete(key);
    }
  }

  function postAI(key, endpoint, text) {
    return request(key, endpoint, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ text }) });
  }

  function hideSuggestions() {
    $("#suggestions-area").hidden = true;
    $("#suggestions-dropdown").replaceChildren();
    $("#suggestions-status").textContent = "";
  }

  function changed() {
    assistantVersion += 1;
    clearTimeout(debounce);
    for (const key of ["suggest", "recommend", "ideas", "art"]) cancel(key);
    $("#char-count").textContent = `${input.value.length} / 1000`;
    $("#input-error").hidden = true;
    input.removeAttribute("aria-invalid");
    hideSuggestions();
    $("#variations-panel").hidden = true;
    $("#recommendation-result").hidden = true;
    $("#assistant-status").textContent = "";
    $("#output-status").textContent = currentArt ? "Текст изменён. Создайте надпись, чтобы обновить результат." : "Введите текст и создайте надпись.";
    if (validText(input.value) && input.value.trim().length >= 3) debounce = setTimeout(suggestions, 300);
  }

  function chooseText(text, banner) {
    input.value = text;
    if (banner && banners.has(banner)) form.elements.banners.value = banner;
    changed();
    clearTimeout(debounce);
    input.focus();
  }

  async function suggestions() {
    const text = input.value;
    $("#suggestions-status").textContent = "Подбираем продолжения…";
    try {
      const data = await postAI("suggest", "/api/suggest", text);
      if (input.value !== text) return;
      if (!Array.isArray(data.suggestions)) throw new Error("Не удалось прочитать подсказки.");
      const buttons = data.suggestions.map((value) => {
        const button = document.createElement("button");
        button.type = "button";
        button.className = "suggestion";
        button.textContent = value;
        button.addEventListener("click", () => chooseText(value));
        return button;
      });
      $("#suggestions-dropdown").replaceChildren(...buttons);
      $("#suggestions-area").hidden = false;
      $("#suggestions-status").textContent = "Выберите вариант или продолжайте писать.";
    } catch (error) {
      if (error.name !== "AbortError" && input.value === text) $("#suggestions-status").textContent = error.message;
    }
  }

  async function assistantAction(kind) {
    if (!validate()) { input.focus(); return; }
    const version = ++assistantVersion;
    cancel("recommend");
    cancel("ideas");
    const text = input.value;
    const button = kind === "recommend" ? recommendButton : ideasButton;
    button.disabled = true;
    button.setAttribute("aria-busy", "true");
    $("#assistant-status").textContent = kind === "recommend" ? "Подбираем стиль…" : "Готовим идеи…";
    try {
      const data = await postAI(kind, kind === "recommend" ? "/api/recommend-banner" : "/api/variations", text);
      if (input.value !== text || assistantVersion !== version) return;
      if (kind === "recommend") {
        if (!banners.has(data.recommended)) throw new Error("Не удалось подобрать стиль.");
        form.elements.banners.value = data.recommended;
        $("#recommendation-result").textContent = `${data.recommended}: ${data.reasoning}`;
        $("#recommendation-result").hidden = false;
        if (currentArt) $("#output-status").textContent = "Стиль выбран. Нажмите «Создать ASCII-арт», чтобы применить его.";
      } else {
        if (!Array.isArray(data.variations)) throw new Error("Не удалось прочитать идеи.");
        const choices = data.variations.map((value) => {
          const button = document.createElement("button");
          button.type = "button";
          button.className = "variation";
          const title = document.createElement("strong");
          title.textContent = value.text;
          const caption = document.createElement("span");
          caption.textContent = `${value.description} · ${value.suggested_banner}`;
          button.append(title, caption);
          button.addEventListener("click", () => chooseText(value.text, value.suggested_banner));
          return button;
        });
        $("#variations-panel").replaceChildren(...choices);
        $("#variations-panel").hidden = false;
      }
      $("#assistant-status").textContent = kind === "recommend" ? "Стиль выбран." : "Выберите идею, чтобы перенести её в текст.";
    } catch (error) {
      if (error.name !== "AbortError" && input.value === text && assistantVersion === version) $("#assistant-status").textContent = error.message;
    } finally {
      button.disabled = false;
      button.removeAttribute("aria-busy");
    }
  }

  function saveHistory() {
    try { localStorage.setItem(historyKey, JSON.stringify(history)); }
    catch { storageAvailable = false; }
  }

  function renderHistory() {
    const items = history.map((item) => {
      const button = document.createElement("button");
      button.type = "button";
      button.className = "history-item";
      button.textContent = `${item.text} · ${item.banner}`;
      button.title = button.textContent;
      button.addEventListener("click", () => { chooseText(item.text, item.banner); form.requestSubmit(); });
      return button;
    });
    $("#history-list").replaceChildren(...items);
    $("#history-empty").textContent = storageAvailable ? "Последние 8 надписей сохраняются только в этом браузере." : "Хранилище недоступно: история сохранится только до закрытия страницы.";
    $("#clear-history").hidden = history.length === 0;
  }

  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    if (!validate()) { input.focus(); return; }
    const version = ++artVersion;
    const text = input.value;
    const banner = form.elements.banners.value;
    generateButton.disabled = true;
    generateButton.setAttribute("aria-busy", "true");
    $("#output-canvas").setAttribute("aria-busy", "true");
    $("#output-status").textContent = "Создаём надпись…";
    try {
      const data = await request("art", "/symbol-art", { method: "POST", body: new URLSearchParams({ inputWord: text, banners: banner }) });
      if (input.value !== text || form.elements.banners.value !== banner) return;
      if (typeof data.art !== "string" || !banners.has(data.banner)) throw new Error("Не удалось прочитать результат.");
      currentArt = data.art;
      currentBanner = data.banner;
      output.textContent = currentArt;
      output.hidden = false;
      $("#empty-state").hidden = true;
      $("#output-label").textContent = currentBanner;
      $("#output-status").textContent = "Готово. Сохраните или скопируйте результат.";
      copyButton.disabled = downloadButton.disabled = false;
      history = [{ text, banner }, ...history.filter((item) => item.text !== text || item.banner !== banner)].slice(0, 8);
      saveHistory();
      renderHistory();
      output.focus({ preventScroll: true });
    } catch (error) {
      if (error.name !== "AbortError") $("#output-status").textContent = error.message;
    } finally {
      if (version === artVersion) {
        generateButton.disabled = false;
        generateButton.removeAttribute("aria-busy");
        $("#output-canvas").setAttribute("aria-busy", "false");
      }
    }
  });

  input.addEventListener("input", changed);
  input.addEventListener("keydown", (event) => { if (event.key === "Escape") { clearTimeout(debounce); cancel("suggest"); hideSuggestions(); } });
  form.querySelectorAll("input[name=banners]").forEach((radio) => radio.addEventListener("change", () => {
    cancel("art");
    cancel("recommend");
    $("#recommendation-result").hidden = true;
    $("#output-status").textContent = currentArt ? "Стиль изменён. Создайте надпись, чтобы обновить результат." : "Стиль выбран. Создайте надпись.";
  }));
  recommendButton.addEventListener("click", () => assistantAction("recommend"));
  ideasButton.addEventListener("click", () => assistantAction("ideas"));
  $("#example-button").addEventListener("click", () => { chooseText("Hello, world!", "standard"); form.requestSubmit(); });

  copyButton.addEventListener("click", async () => {
    let timer;
    $("#output-status").textContent = "Копируем…";
    try {
      await Promise.race([
        navigator.clipboard.writeText(currentArt),
        new Promise((_, reject) => { timer = setTimeout(() => reject(new Error("Буфер обмена недоступен")), 1500); }),
      ]);
      $("#output-status").textContent = "Скопировано. Вставляйте в блок с моноширинным шрифтом.";
    } catch {
      $("#output-status").textContent = "Браузер не разрешил копирование. Результат выделен: нажмите Ctrl+C (⌘C на Mac) или скачайте .txt.";
      output.focus();
      const selection = window.getSelection();
      const range = document.createRange();
      range.selectNodeContents(output);
      selection?.removeAllRanges();
      selection?.addRange(range);
    } finally {
      clearTimeout(timer);
    }
  });
  downloadButton.addEventListener("click", () => {
    const url = URL.createObjectURL(new Blob([currentArt], { type: "text/plain;charset=utf-8" }));
    const link = document.createElement("a");
    link.href = url;
    link.download = `symbol-${currentBanner}.txt`;
    document.body.append(link);
    link.click();
    link.remove();
    setTimeout(() => URL.revokeObjectURL(url), 1000);
    $("#output-status").textContent = "Файл .txt подготовлен к скачиванию.";
  });
  $("#clear-history").addEventListener("click", () => { history = []; saveHistory(); renderHistory(); });

  try {
    const saved = JSON.parse(localStorage.getItem(historyKey) || "[]");
    if (Array.isArray(saved)) history = saved.filter((item) => item && typeof item.text === "string" && validText(item.text) && banners.has(item.banner)).slice(0, 8);
  } catch { storageAvailable = false; }
  recommendButton.disabled = ideasButton.disabled = $("#example-button").disabled = false;
  copyButton.disabled = downloadButton.disabled = !currentArt;
  $("#char-count").textContent = `${input.value.length} / 1000`;
  renderHistory();
})();
