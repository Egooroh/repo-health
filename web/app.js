'use strict';

const main = document.querySelector('#main');
const dialog = document.querySelector('#analysis-dialog');
const state = { demo: sessionStorage.getItem('repo-health-demo') === 'true', items: [], search: '', serial: 0, timer: null };
const esc = value => String(value ?? '').replace(/[&<>"']/g, c => ({ '&':'&amp;', '<':'&lt;', '>':'&gt;', '"':'&quot;', "'":'&#39;' }[c]));
const num = value => value == null ? '—' : new Intl.NumberFormat('ru-RU', {maximumFractionDigits:1}).format(value);
const date = value => value ? new Intl.DateTimeFormat('ru-RU', {day:'numeric',month:'short',hour:'2-digit',minute:'2-digit'}).format(new Date(value)) : 'Ещё не анализировался';
const scoreClass = score => score == null ? '' : score >= 80 ? 'score-green' : score >= 50 ? 'score-amber' : 'score-red';
const scoreLabel = score => score == null ? 'Недостаточно данных' : score >= 80 ? 'Хорошее состояние' : score >= 50 ? 'Есть что улучшить' : 'Требует внимания';
const isPending = job => job && ['queued','running'].includes(job.status);
const route = () => (location.hash.replace(/^#\/?/, '') || 'repositories').split('/');
let toastTimer;

async function api(path, body) {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 15000);
  try {
    const response = await fetch(`/api/v1${path}`, { signal:controller.signal, ...(body === undefined ? {} : {method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)}) });
    const data = await response.json();
    if (!response.ok) throw new Error(data.error || 'Не удалось выполнить запрос.');
    return data;
  } catch (error) {
    if (error.name === 'AbortError') throw new Error('Сервер не ответил вовремя. Повторите попытку.');
    if (error instanceof TypeError) throw new Error('Нет соединения с приложением. Проверьте, запущен ли сервер.');
    throw error;
  } finally { clearTimeout(timeout); }
}

function toast(message) {
  const element = document.querySelector('#toast');
  element.textContent = message; element.hidden = false;
  clearTimeout(toastTimer); toastTimer = setTimeout(() => { element.hidden = true; }, 4500);
}

function modeUI() {
  document.querySelector('#mode-banner').hidden = !state.demo;
  document.querySelector('#demo-toggle').innerHTML = state.demo ? 'Реальные репозитории <span aria-hidden="true">↗</span>' : 'Посмотреть демо <span aria-hidden="true">↗</span>';
  sessionStorage.setItem('repo-health-demo', String(state.demo));
}

async function toggleDemo() {
  const button = document.querySelector('#demo-toggle'); button.disabled = true;
  try {
    if (!state.demo) await api('/demo', {});
    state.demo = !state.demo; modeUI(); state.search = '';
    if (location.hash === '#/repositories') await load(); else location.hash = '#/repositories';
  } catch(error) { toast(error.message); } finally { button.disabled = false; }
}

document.querySelector('#demo-toggle').addEventListener('click', toggleDemo);
document.querySelector('#exit-demo').addEventListener('click', toggleDemo);
document.querySelector('#close-dialog').addEventListener('click', () => dialog.close());
dialog.addEventListener('click', event => { if(event.target === dialog) { const rect=dialog.getBoundingClientRect(); if(event.clientX<rect.left || event.clientX>rect.right || event.clientY<rect.top || event.clientY>rect.bottom) dialog.close(); } });

function openAnalysis() {
  document.querySelector('#form-error').hidden = true;
  dialog.showModal(); document.querySelector('#repository-ref').focus();
}

document.querySelector('#analysis-form').addEventListener('submit', async event => {
  event.preventDefault();
  const button = document.querySelector('#submit-analysis'); const errorBox = document.querySelector('#form-error');
  button.disabled = true; errorBox.hidden = true;
  try {
    const job = await api('/analyses', {ref:document.querySelector('#repository-ref').value, demo:false});
    state.demo = false; modeUI(); dialog.close(); event.target.reset();
    location.hash = `#/repository/${job.repository_id}`;
    toast('Анализ поставлен в очередь. Результат появится на этой странице.');
    await load();
  } catch(error) { errorBox.textContent = error.message; errorBox.hidden = false; }
  finally { button.disabled = false; }
});

document.addEventListener('click', async event => {
  const target = event.target.closest('[data-action]'); if (!target) return;
  const action = target.dataset.action;
  if (action === 'open-analysis') openAnalysis();
  if (action === 'demo') await toggleDemo();
  if (action === 'reload') await load();
  if (action === 'reanalyze') {
    target.disabled = true;
    try { await api('/analyses', {ref:target.dataset.ref,demo:target.dataset.demo === 'true'}); toast('Повторный анализ запущен.'); await load(); }
    catch(error) { toast(error.message); target.disabled = false; }
  }
});

main.addEventListener('input', event => {
  if(event.target.id === 'search') { state.search = event.target.value; renderCollection(); }
});

window.addEventListener('hashchange', () => {state.search = '';load();});

async function load() {
  clearTimeout(state.timer);
  const serial = ++state.serial;
  const [page,id] = route();
  const title = ({repositories:'Репозитории',leaderboard:'Рейтинг',methodology:'Методика оценки',repository:'Аналитика репозитория'})[page] || 'Страница не найдена';
  document.querySelector('#breadcrumb').textContent = title;
  document.title = `${title} — Repo Health`;
  document.querySelectorAll('[data-nav]').forEach(link => {const active=link.dataset.nav === (page === 'repository' ? 'repositories' : page);link.classList.toggle('active',active);if(active)link.setAttribute('aria-current','page');else link.removeAttribute('aria-current');});
  modeUI();
  try {
    if (page === 'methodology') { main.innerHTML = methodology(); return; }
    if (page === 'repository' && id) {
      const project = await api(`/repositories/${encodeURIComponent(id)}`);
      if (serial !== state.serial) return;
      if(state.demo !== project.demo) { state.demo=project.demo;modeUI(); }
      main.innerHTML = detail(project);
      if(isPending(project.job)) state.timer = setTimeout(load, 1200);
    } else if (page === 'repositories' || page === 'leaderboard') {
      const data = await api(`/${page}?demo=${state.demo}`);
      if(serial !== state.serial) return;
      const searchFocused = document.activeElement?.id === 'search';
      const searchSelection = searchFocused ? [document.activeElement.selectionStart, document.activeElement.selectionEnd] : null;
      state.items = data.items;
      main.innerHTML = collectionPage(page);
      renderCollection();
      if(searchFocused && document.querySelector('#search')) {
        document.querySelector('#search').focus({preventScroll:true});
        document.querySelector('#search').setSelectionRange(...searchSelection);
      }
      if(state.items.some(p => isPending(p.job))) state.timer = setTimeout(load, 1200);
    } else {main.innerHTML = `<div class="empty"><h3>Страница не найдена</h3><a class="button button-primary" href="#/repositories">К репозиториям</a></div>`;}
  } catch(error) {
    if(serial !== state.serial) return;
    main.innerHTML = `<div class="error-state" role="alert"><span class="eyebrow">НЕ УДАЛОСЬ ЗАГРУЗИТЬ</span><h2>Попробуем ещё раз?</h2><p>${esc(error.message)}</p><button class="button button-primary" data-action="reload">Повторить</button></div>`;
  }
}

function collectionPage(page) {
  const ranking = page === 'leaderboard';
  const scored = state.items.filter(p => p.report?.score != null);
  const average = scored.length ? scored.reduce((n,p) => n + p.report.score, 0) / scored.length : null;
  const recommendations = state.items.reduce((n,p) => n + (p.report?.recommendations.length || 0), 0);
  return `<div class="page-heading"><div><span class="eyebrow">${ranking ? 'ОТКРЫТЫЕ ПРОЕКТЫ' : 'ОБЗОР РАБОЧЕГО ПРОСТРАНСТВА'}</span><h1>${ranking ? 'Рейтинг репозиториев' : 'Ваши репозитории'}</h1><p>${ranking ? 'Открытые проекты с актуальной оценкой и достаточными данными.' : 'Состояние проекта, причины оценки и следующий шаг — в одном месте.'}</p></div><span class="tag ${state.demo?'amber':'green'}">${state.demo?'Вымышленные примеры':'Открытые репозитории SourceCraft'}</span></div>
    <section class="stats" aria-label="Краткая статистика"><div class="stat"><div class="stat-label">${ranking?'В рейтинге':'Всего репозиториев'} <span aria-hidden="true">▦</span></div><div class="stat-number">${state.items.length}</div><div class="stat-note">${state.demo?'Демонстрационные данные':'В этом экземпляре сервиса'}</div></div><div class="stat"><div class="stat-label">Средняя оценка <span aria-hidden="true">↗</span></div><div class="stat-number">${num(average)} <small>/ 100</small></div><div class="stat-note">По ${scored.length} рассчитанным отчётам</div></div><div class="stat"><div class="stat-label">Рекомендации <span aria-hidden="true">≡</span></div><div class="stat-number">${recommendations}</div><div class="stat-note">Конкретные действия для улучшения</div></div></section>
    ${!ranking && !state.items.length ? welcome() : ''}
    ${ranking ? '<div class="notice">В рейтинг попадают проанализированные непустые репозитории с полнотой данных от 60% и оценкой не старше 7 дней. Сортировка: балл → полнота данных → название. Это рейтинг проектов, добавленных в этот сервис.</div>' : ''}
    <div class="section-heading"><h2>${ranking ? 'Участники рейтинга' : 'Все проекты'} <span class="count">${state.items.length}</span></h2>${state.items.length?`<input class="search" id="search" aria-label="Поиск по названию или языку" placeholder="Поиск по названию или языку…" value="${esc(state.search)}">`:''}</div><div id="collection"></div>`;
}

function welcome() {
  return `<section class="hero"><div class="hero-content"><span class="eyebrow">ОТ ДАННЫХ К ДЕЙСТВИЯМ</span><h2>Узнайте, что работает.<br>Поймите, что улучшить.</h2><p>Добавьте репозиторий SourceCraft. Мы проверим документацию, CI и работу с задачами — и объясним каждый балл.</p><button class="button" data-action="open-analysis">Добавить первый репозиторий <span aria-hidden="true">→</span></button></div><div class="hero-visual" aria-hidden="true"><div class="hero-orbit"><svg class="hero-pulse" viewBox="0 0 100 65"><path d="M0 34h24l12-21 15 40 13-28 9 9h27"/></svg><span class="hero-label">Факты → оценка → действия</span></div></div></section>`;
}

function renderCollection() {
  const container = document.querySelector('#collection'); if(!container) return;
  const items = state.items.filter(p => `${p.ref} ${p.report?.repository.language || ''}`.toLowerCase().includes(state.search.toLowerCase()));
  if(!items.length) {
    container.innerHTML = `<div class="empty"><h3>${state.items.length?'Ничего не найдено':route()[0]==='leaderboard'?'Рейтинг пока формируется':'Здесь появятся ваши проекты'}</h3><p>${state.items.length?'Попробуйте другое название или язык.':'Добавьте открытый репозиторий или познакомьтесь с сервисом на демонстрационных примерах.'}</p>${!state.items.length?`<button class="button button-soft" data-action="${state.demo?'open-analysis':'demo'}">${state.demo?'Анализировать реальный репозиторий':'Посмотреть демо →'}</button>`:''}</div>${!state.items.length && route()[0]!=='leaderboard'?'<section class="steps"><div class="step"><span class="step-no">01</span><h3>Добавьте репозиторий</h3><p>Достаточно адреса открытого проекта SourceCraft.</p></div><div class="step"><span class="step-no">02</span><h3>Изучите оценку</h3><p>У каждого показателя есть источник и объяснение.</p></div><div class="step"><span class="step-no">03</span><h3>Выберите следующий шаг</h3><p>Рекомендации связаны с обнаруженными фактами.</p></div></section>':''}`;
    return;
  }
  if(route()[0] === 'leaderboard') {
    container.innerHTML = `<div class="table-wrap"><table><thead><tr><th>Место</th><th>Репозиторий</th><th>Оценка</th><th>Полнота данных</th><th>Последний анализ</th><th><span class="muted">Отчёт</span></th></tr></thead><tbody>${items.map(p=>{const index=state.items.indexOf(p);return `<tr><td><span class="rank ${index===0?'first':''}">${String(index+1).padStart(2,'0')}</span></td><td><a class="table-repo" href="#/repository/${p.id}">${esc(p.ref)}<small>${esc(p.report.repository.language || 'Язык не указан')}${p.demo?' · Демо':''}</small></a></td><td><strong class="${scoreClass(p.report.score)}">${num(p.report.score)}</strong><span class="muted"> / 100</span></td><td>${p.report.coverage}%</td><td class="muted">${date(p.report.analyzed_at)}</td><td><a class="text-link" href="#/repository/${p.id}" aria-label="Отчёт ${esc(p.ref)}">Подробнее ↗</a></td></tr>`}).join('')}</tbody></table></div>`;
  } else {container.innerHTML = `<div class="cards">${items.map(card).join('')}</div>`;}
}

function statusTag(p) {
  if(isPending(p.job)) return '<span class="tag green"><span class="pulse-dot"></span>Анализируем</span>';
  if(p.job?.status === 'failed') return '<span class="tag red">Ошибка анализа</span>';
  if(p.report?.score == null) return '<span class="tag gray">Мало данных</span>';
  return `<span class="tag ${p.report.score>=80?'green':'amber'}">${p.report.score>=80?'В порядке':'Есть рекомендации'}</span>`;
}

function card(p) {
  const [owner,name] = p.ref.split('/'); const report = p.report;
  return `<article class="repo-card"><div class="repo-card-top"><span class="repo-icon" aria-hidden="true">⌘</span>${statusTag(p)}</div><h3><a href="#/repository/${p.id}"><span>${esc(owner)} /</span>${esc(name)}</a></h3><p class="repo-description">${esc(report?.repository.description || (isPending(p.job)?'Собираем сведения о репозитории. Страница обновится автоматически.':p.job?.error || 'Описание не указано.'))}</p><div class="card-score"><div><div class="score-text ${scoreClass(report?.score)}">${num(report?.score)} <small>/ 100</small></div><div class="score-caption">${report?.coverage<60?'Предварительная оценка':'Оценка здоровья'}</div></div><div><span class="tag">${report?report.coverage+'% данных':'Ожидает анализа'}</span></div></div><div class="card-footer"><span>${esc(report?.repository.language || 'Язык не указан')}${p.demo?' · Демо':''}</span><a class="text-link" href="#/repository/${p.id}">Подробнее ↗</a></div></article>`;
}

function detail(p) {
  const report = p.report; const [owner,name] = p.ref.split('/');
  const pending = isPending(p.job);
  const header = `<a class="back-link" href="#/repositories">← Все репозитории</a><div class="page-heading repo-heading"><div><span class="eyebrow">${p.demo?'ДЕМОНСТРАЦИОННЫЙ ПРИМЕР':'АНАЛИТИКА РЕПОЗИТОРИЯ'}</span><h1><span>${esc(owner)} /</span> ${esc(name)}</h1><p>${esc(report?.repository.description || 'Результат анализа открытого репозитория SourceCraft.')}</p><div class="repo-submeta">${report?`<span class="tag">${esc(report.repository.language || 'Язык не указан')}</span><span class="tag">${esc(report.repository.default_branch || 'Ветка не указана')}</span><span class="tag">${date(report.analyzed_at)}</span>`:''}${report?.repository.fork?'<span class="tag amber">Форк</span>':''}${!p.demo?`<a class="tag" href="https://sourcecraft.dev/${esc(p.ref)}" target="_blank" rel="noopener noreferrer">Открыть в SourceCraft ↗</a>`:''}</div></div><button class="button button-soft" data-action="reanalyze" data-ref="${esc(p.ref)}" data-demo="${p.demo}" ${pending?'disabled':''}>${pending?'Анализируем…':'↻ Обновить оценку'}</button></div>`;
  const progress = pending?`<div class="notice" role="status"><span class="pulse-dot"></span>${p.job.status==='queued'?'Анализ в очереди.':'Собираем данные и рассчитываем оценку.'} Страница обновится автоматически.${report?' Пока показан предыдущий сохранённый результат.':''}</div>`:'';
  const failure = p.job?.status==='failed'?`<div class="notice error" role="alert"><strong>Анализ не завершён.</strong> ${esc(p.job.error)}${report?' Ниже — последний успешный результат.':''}</div>`:'';
  if(!report) return `${header}${progress}${failure}${!pending?'<div class="empty"><h3>Готового отчёта пока нет</h3><p>Устраните причину ошибки и запустите анализ повторно.</p></div>':'<div class="loading" role="status">Проверяем документацию, CI и задачи…</div>'}`;
  const offset = 440-(report.score??0)/100*440;
  return `${header}${progress}${failure}${!report.eligible?`<div class="notice amber">${esc(report.eligibility_reason)} Репозиторий не включён в рейтинг.</div>`:''}
    <div class="detail-grid"><section class="panel score-panel"><h2 class="panel-title">${report.coverage<60?'Предварительная оценка':'Оценка здоровья'}</h2><div class="score-ring"><svg viewBox="0 0 160 160" aria-hidden="true"><circle class="ring-bg" cx="80" cy="80" r="70"/><circle class="ring-value ${report.score!=null&&report.score<50?'red':report.score!=null&&report.score<80?'amber':''}" cx="80" cy="80" r="70" stroke-dasharray="440" stroke-dashoffset="${offset}"/></svg><div class="ring-text"><strong>${num(report.score)}</strong><small>из 100 баллов</small></div></div><span class="tag ${report.score==null?'gray':report.score>=80?'green':'amber'}">${scoreLabel(report.score)}</span><p class="score-explainer">Оценка по доступным сигналам.<br>Каждый показатель можно раскрыть справа.</p><div class="coverage"><div class="coverage-line"><span>Полнота данных</span><strong>${report.coverage}%</strong></div><meter class="bar" min="0" max="100" value="${report.coverage}" aria-label="Полнота данных ${report.coverage}%"></meter><p>Доля весов с измеренными показателями. Не вероятность правильности оценки.</p></div></section>
    <section class="panel"><h2 class="panel-title">Из чего складывается оценка</h2>${report.categories.map(category).join('')}</section></div>
    <section class="panel detail-lower"><div class="section-heading"><h2>Что улучшить в первую очередь <span class="count">${report.recommendations.length}</span></h2><span class="muted small-label">НА ОСНОВЕ ФАКТОВ</span></div>${report.recommendations.length?report.recommendations.map((r,i)=>`<article class="recommendation"><span class="rec-index">${String(i+1).padStart(2,'0')}</span><div><h3>${esc(r.title)}</h3><p>${esc(r.reason)}</p><p class="action">${esc(r.action)}</p></div><span class="tag ${r.priority==='high'?'amber':'gray'}">${r.priority==='high'?'Высокий приоритет':'Средний приоритет'}</span></article>`).join(''):`<div class="notice">${report.score==null?'Недостаточно данных, чтобы дать содержательные рекомендации.':'По измеренным показателям рекомендаций нет. Неизвестные источники, включая AppSec, не позволяют сделать вывод о состоянии всего проекта.'}</div>`}</section>
    <section class="panel detail-lower"><h2 class="panel-title">Источники и ограничения</h2><div class="sources">${Object.entries(report.sources).map(([key,source])=>`<div class="source"><strong>${esc(({files:'Документы',ci:'История CI',issues:'Issues',pulls:'Merge Requests',appsec:'AppSec'})[key] || key)}</strong><span class="tag ${source.status==='available'?'green':'amber'}">${({available:'Доступно',unavailable:'Недоступно',truncated:'Неполные данные',not_connected:'Не подключено'})[source.status] || 'Неизвестно'}</span><p>${esc(source.detail)}</p></div>`).join('')}</div><p class="score-explainer">Методика ${esc(report.version)}. Наличие файла не означает высокое качество содержимого. Для форков и зрелых проектов интерпретируйте результат с учётом назначения. <a class="text-link" href="#/methodology">Как считается оценка →</a></p></section>`;
}

function category(c) {
  return `<div class="category"><div class="category-line"><strong>${esc(c.title)} <small>· вес ${c.weight}%</small></strong><span class="${scoreClass(c.score)}">${c.score==null?'Нет данных':num(c.score)+' / 100'}</span></div><meter class="bar" min="0" max="100" value="${c.score??0}" aria-label="${esc(c.title)}: ${c.score==null?'нет данных':num(c.score)}"></meter><details><summary>Показатели и объяснение</summary>${c.metrics.map(m=>`<div class="metric-detail"><strong>${esc(m.title)} · ${m.score==null?'Неизвестно':num(m.score)+' / 100'} · вес ${m.weight}%</strong>${esc(m.evidence)}</div>`).join('')}</details></div>`;
}

function methodology() {
  return `<div class="methodology"><div class="page-heading"><div><span class="eyebrow">ПРОЗРАЧНАЯ МЕТОДИКА · MVP-1.0</span><h1>За каждым баллом — факт</h1><p>Простая воспроизводимая оценка. Известные ограничения. Понятные действия.</p></div></div><section class="panel"><h2>Как получается результат</h2><p>Сервис собирает факты из SourceCraft и применяет фиксированные правила. Языковая модель в расчёте не используется. Одинаковые данные и момент анализа дают одинаковый результат.</p><div class="formula">Оценка = сумма (балл показателя × его вес) / сумма весов известных показателей</div><p>Пример: документация — 100 баллов с весом 30, CI — 60 с весом 30; остальные данные неизвестны. Итог: (100 × 30 + 60 × 30) / 60 = <strong>80 баллов</strong>. Полнота данных — <strong>60%</strong>.</p><p>Если измеренных показателей нет или репозиторий пуст, численная оценка не выставляется. Недоступные данные не означают ни успех, ни проблему.</p></section>
  <section class="panel"><h2>Что измеряет базовая версия</h2><div class="table-wrap"><table><thead><tr><th>Показатель</th><th>Вес</th><th>Правило</th></tr></thead><tbody><tr><td>README</td><td>15%</td><td>100 при наличии, иначе 0.</td></tr><tr><td>LICENSE / COPYING</td><td>10%</td><td>100 при наличии, иначе 0.</td></tr><tr><td>CONTRIBUTING</td><td>5%</td><td>100 при наличии, иначе 0.</td></tr><tr><td>Надёжность CI</td><td>30%</td><td>Доля success среди success, failed и timeout за 30 дней. Отменённые, пропущенные и незавершённые запуски исключены. Нет завершённых запусков — нет оценки.</td></tr><tr><td>Очередь issues</td><td>20%</td><td>100 × (1 − доля незавершённых issues без обновлений более 30 дней). Если незавершённых issues нет, 100.</td></tr><tr><td>Запросы на слияние</td><td>10%</td><td>Та же формула для open и merging с порогом 14 дней. Черновики и закрытые запросы исключены.</td></tr><tr><td>AppSec</td><td>10%</td><td>Не подключён. Всегда неизвестно, в расчёт не входит.</td></tr></tbody></table></div><p>Документы ищутся в корне, docs, .sourcecraft и .github, без анализа качества текста. Источники читаются до конца, максимум 5 страниц по 100 записей каждый. При превышении лимита весь соответствующий источник исключается из оценки.</p></section>
  <section class="panel"><h2>Полнота данных и рейтинг</h2><p>Полнота данных — сумма весов измеренных показателей. Максимум текущей версии — 90%, поскольку AppSec не подключён. Показатель не является вероятностью корректности оценки.</p><p>В рейтинг входят непустые репозитории с полнотой от 60%, текущей версией методики и анализом не старше 7 дней. При равных баллах сначала сравнивается полнота, затем название. Демо и реальные проекты всегда разделены. Рейтинг охватывает добавленные в этот экземпляр сервиса проекты.</p></section>
  <section class="panel"><h2>Где заканчиваются выводы сервиса</h2><ul><li>Это базовая инженерная проверка, а не аудит безопасности или качества всего исходного кода.</li><li>Наличие README и лицензии не оценивает их содержание и применимость.</li><li>Малая активность коммитов не штрафуется. Стабильный проект может редко изменяться.</li><li>Обновление issue само по себе не доказывает решение задачи. Метрика чувствительна к формальным изменениям.</li><li>Порогам и весам нужна дальнейшая калибровка на проектах разных типов.</li><li>Рекомендации формируются по наблюдаемым фактам. Обещаний прибавки баллов нет.</li><li>Результат показывает состояние на момент последнего анализа. Автоматический пересчёт по расписанию в MVP не включён.</li></ul><a class="text-link" href="https://sourcecraft.dev/portal/docs/en/sourcecraft/operations/api-start" target="_blank" rel="noopener noreferrer">Официальная документация SourceCraft API ↗</a></section></div>`;
}

load();
