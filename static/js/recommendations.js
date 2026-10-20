(() => {
  // [Fix] id кнопки как в ТЗ: #ai-recommend-btn (старый #recommend-button тоже поддерживаем).
  const recommendationButton = document.querySelector('#ai-recommend-btn, #recommend-button');
  const recommendationResult = document.querySelector('#recommendation-result');
  const assistantStatus = document.querySelector('#assistant-status');
  const input = document.querySelector('#text-input');

  if (!recommendationButton || !recommendationResult || !assistantStatus || !input) {
    console.error('recommendations.js: required elements not found');
    return;
  }

  const buttonLabel = recommendationButton.innerHTML;

  // The server answers with "standard" / "shadow" / "thinkertoy",
  // but the radio buttons in index.html have the values "standard.txt", etc.
  function bannerRadio(name) {
    return document.querySelector(`input[name="banners"][value="${name}.txt"]`);
  }

  function selectBanner(name) {
    const radio = bannerRadio(name);
    if (radio) {
      radio.checked = true;
      radio.dispatchEvent(new Event('change', { bubbles: true }));
    }
  }

  // [Fix] ТЗ: «Highlight recommended banner in UI» — подсвечиваем карточку
  // рекомендованного баннера, пока открыт ответ (раньше подсветки не было).
  function highlightBanner(name) {
    document.querySelectorAll('.banner-card.is-recommended')
      .forEach((card) => card.classList.remove('is-recommended'));
    const radio = name ? bannerRadio(name) : null;
    if (radio && radio.closest('.banner-card')) {
      radio.closest('.banner-card').classList.add('is-recommended');
    }
  }

  // [Fix] ТЗ: «Show loading state during API call» — меняем текст кнопки.
  function setLoading(loading) {
    recommendationButton.disabled = loading; // prevent double clicks while waiting
    recommendationButton.setAttribute('aria-busy', String(loading));
    if (loading) {
      recommendationButton.textContent = 'Подбираем…';
      assistantStatus.textContent = 'Анализируем текст…';
    } else {
      recommendationButton.innerHTML = buttonLabel;
    }
  }

  // Build elements with textContent (no innerHTML), so server text can't inject markup.
  function el(tag, text, className) {
    const node = document.createElement(tag);
    if (text !== undefined) node.textContent = text;
    if (className) node.className = className;
    return node;
  }

  function closeRecommendation() {
    recommendationResult.hidden = true;
    recommendationResult.replaceChildren();
    assistantStatus.textContent = '';
    highlightBanner(null);
  }

  function renderResult(data) {
    const alternatives = Array.isArray(data.alternatives) ? data.alternatives : [];

    const list = el('ul');
    alternatives.forEach((alt) => {
      const item = el('li');
      item.append(
        el('strong', alt.banner),
        el('p', `Score: ${alt.score}`),
        el('p', `Reason: ${alt.reason}`)
      );
      list.append(item);
    });

    // Buttons: the user decides whether to apply the style or just close the box
    const applyButton = el('button', 'Use this style', 'text-button');
    applyButton.type = 'button';
    applyButton.addEventListener('click', () => {
      selectBanner(data.recommended);
      closeRecommendation();
    });

    const closeButton = el('button', 'Close', 'text-button');
    closeButton.type = 'button';
    closeButton.addEventListener('click', closeRecommendation);

    const actions = el('div');
    actions.style.display = 'flex';
    actions.style.gap = '16px';
    actions.append(applyButton, closeButton);

    recommendationResult.replaceChildren(
      el('h3', 'Recommended Banner'),
      el('p', data.recommended),
      el('h3', 'Reasoning'),
      el('p', data.reasoning),
      el('h3', 'Alternatives'),
      list,
      actions
    );
    recommendationResult.hidden = false;
    highlightBanner(data.recommended);
  }

  recommendationButton.addEventListener('click', async () => {
    assistantStatus.textContent = '';
    recommendationResult.hidden = true;
    highlightBanner(null);

    if (input.value.trim() === '') {
      assistantStatus.textContent = 'Enter some text first.';
      return;
    }

    setLoading(true);

    try {
      // [Fix] Формат запроса из ТЗ: JSON {"text": "..."} (раньше — форма inputWord=...).
      const response = await fetch('/api/recommend-banner', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
        body: JSON.stringify({ text: input.value }),
      });

      // [Fix] Ошибки сервер теперь отдаёт как JSON {"error": "..."}.
      let data = null;
      try {
        data = await response.json();
      } catch {
        data = null;
      }

      if (!response.ok) {
        assistantStatus.textContent =
          (data && data.error) || 'Could not get a recommendation.';
        return;
      }
      if (!data || typeof data.recommended !== 'string') {
        assistantStatus.textContent = 'The server returned an invalid recommendation.';
        return;
      }

      assistantStatus.textContent = '';
      renderResult(data);
      // NOTE: the selected banner is NOT changed here anymore.
    } catch (err) {
      console.error(err);
      assistantStatus.textContent = 'Network error. Please try again.';
    } finally {
      setLoading(false);
    }
  });

  // Text changed -> the old recommendation is no longer relevant.
  input.addEventListener('input', () => {
    if (!recommendationResult.hidden) closeRecommendation();
  });
})();
