// Shows why the app did not start instead of leaving a blank page.
// Classic script (not a module) so it also runs on browsers that cannot load the app.
(function () {
  var shown = false
  function show(reason) {
    if (shown) return
    shown = true
    var box = document.createElement('div')
    box.setAttribute('role', 'alert')
    box.style.cssText = 'padding:24px 20px;font:15px/1.7 -apple-system,sans-serif;color:#1f1e1d;background:#faf9f5;min-height:100vh'
    var h = document.createElement('p')
    h.style.cssText = 'font-weight:700;margin:0 0 8px'
    h.textContent = 'remote-herdr を表示できませんでした'
    var p = document.createElement('p')
    p.style.cssText = 'margin:0 0 12px'
    p.textContent = '再読み込みしても直らない場合は、下の内容を共有してください。'
    var pre = document.createElement('pre')
    pre.style.cssText = 'white-space:pre-wrap;word-break:break-all;font:12px/1.5 ui-monospace,monospace;background:#f0eee6;padding:12px;border-radius:10px'
    pre.textContent = reason + '\n\n' + navigator.userAgent
    box.appendChild(h)
    box.appendChild(p)
    box.appendChild(pre)
    document.body.appendChild(box)
  }
  window.addEventListener(
    'error',
    function (e) {
      var t = e.target
      if (t && t.tagName === 'SCRIPT') show('スクリプトを読み込めませんでした: ' + (t.src || ''))
      else if (e.message) show(e.message + ' (' + (e.filename || '') + ':' + (e.lineno || '') + ')')
    },
    true,
  )
  window.addEventListener('unhandledrejection', function (e) {
    var r = e.reason
    if (!window.__remoteHerdrStarted) show('起動中のエラー: ' + ((r && r.message) || r))
  })
  setTimeout(function () {
    var app = document.getElementById('app')
    if (!window.__remoteHerdrStarted && app && !app.firstElementChild) show('時間内に起動しませんでした')
  }, 8000)
})()
