(() => {
  const recommendationButton = document.querySelector('#recommend-button');
  const recommendationResult = document.querySelector('#recommendation-result');
  const assistantStatus = document.querySelector('#assistant-status');
  const input = document.querySelector('#text-input');

  if (!recommendationButton || !recommendationResult || !assistantStatus || !input) {
    console.error('recommendations.js: required elements not found');
    return;
  }

  // The server answers with "standard" / "shadow" / "thinkertoy",
  // but the radio buttons in index.html have the values "standard.txt", etc.
  function selectBanner(name) {
    const radio = document.querySelector(`input[name="banners"][value="${name}.txt"]`);
    if (radio) {
      radio.checked = true;
      radio.dispatchEvent(new Event('change', { bubbles: true }));
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
  }

  recommendationButton.addEventListener('click', async () => {
    assistantStatus.textContent = '';
    recommendationResult.hidden = true;

    if (input.value.trim() === '') {
      assistantStatus.textContent = 'Enter some text first.';
      return;
    }

    recommendationButton.disabled = true; // prevent double clicks while waiting

    try {
      const response = await fetch('/api/recommend-banner', {
        method: 'POST',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        body: new URLSearchParams({ inputWord: input.value }),
      });

      if (!response.ok) {
        // HandleBannerRec uses http.Error, so the error body is plain text
        assistantStatus.textContent =
          (await response.text()).trim() || 'Could not get a recommendation.';
        return;
      }

      renderResult(await response.json());
      // NOTE: the selected banner is NOT changed here anymore.
    } catch (err) {
      console.error(err);
      assistantStatus.textContent = 'Network error. Please try again.';
    } finally {
      recommendationButton.disabled = false;
    }
  });
})();