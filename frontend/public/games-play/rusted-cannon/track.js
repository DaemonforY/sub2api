// HiveGPT 小游戏 events (/api/v1/events, same visitor/session ids as the main site; no cookies).
// The game keeps its state in a top-level `State`; we watch its mode to spot starts and finishes.
(function () {
  if (navigator.webdriver) return;
  var GAME = 'rusted-cannon';
  function rid() { var b = new Uint8Array(12); crypto.getRandomValues(b); return Array.from(b, function (x) { return x.toString(16).padStart(2, '0'); }).join(''); }
  function id(store, key, ok) { try { var v = store.getItem(key) || ''; if (!ok(v)) { v = rid(); store.setItem(key, v); } return v; } catch (e) { return rid(); } }
  function attr() {
    try { var saved = JSON.parse(localStorage.getItem('hg_attr') || 'null'); if (saved) return saved; } catch (e) {}
    var q = new URLSearchParams(location.search), ref = '';
    try { if (document.referrer && new URL(document.referrer).host !== location.host) ref = document.referrer.slice(0, 300); } catch (e) {}
    return { source: (q.get('utm_source') || '').slice(0, 64), medium: (q.get('utm_medium') || '').slice(0, 64), campaign: (q.get('utm_campaign') || '').slice(0, 64), aff: (q.get('aff') || '').slice(0, 64), referrer: ref, landing: location.pathname, first_seen: Date.now() };
  }
  function send(name, props) {
    var ev = { name: name, path: location.pathname };
    if (props) ev.props = props;
    var body = JSON.stringify({
      visitor_id: id(localStorage, 'hg_vid', function (v) { return /^[A-Za-z0-9_-]{8,64}$/.test(v); }),
      session_id: id(sessionStorage, 'hg_sid', function (v) { return !!v; }),
      app: 'main', attr: attr(), events: [ev]
    });
    try { fetch('/api/v1/events', { method: 'POST', keepalive: true, headers: { 'Content-Type': 'application/json' }, body: body }).catch(function () {}); } catch (e) {}
  }
  send('page_view');
  var last = '';
  var started = false;
  setInterval(function () {
    var s;
    try { s = State; } catch (e) { return; } // eslint-disable-line no-undef
    if (!s) return;
    document.body.classList.toggle('playing', s.mode === 'PLAY');
    if (s.mode === last) return;
    var prev = last;
    last = s.mode;
    var mission = (s.mission || 0) + 1;
    if (s.mode === 'PLAY' && prev === 'BRIEF') {
      send('game_start', { game: GAME, mission: mission, again: started, lang: window.RC_LANG || 'zh' });
      started = true;
    }
    if (s.mode === 'GAMEOVER') send('game_finish', { game: GAME, kind: 'dead', mission: mission, score: s.score || 0 });
    if (s.mode === 'ENDING') send('game_finish', { game: GAME, kind: 'complete', mission: mission, score: s.score || 0 });
  }, 500);
})();
