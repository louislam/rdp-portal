// End-to-end test: launches the real rdp-portal.exe, drives its WebView2 over
// the devtools protocol (no browser download, no extra dependencies) and checks
// both the UI and the files it writes.
import {spawn} from 'node:child_process'
import {cpSync, existsSync, mkdtempSync, readFileSync, readdirSync, rmSync} from 'node:fs'
import {tmpdir} from 'node:os'
import {join} from 'node:path'

const PORT = 9333
const EXE = join(process.cwd(), 'bin', 'RDP-Portal.exe')

let passed = 0
const failures = []
function check(name, cond, detail = '') {
  if (cond) {
    passed++
    console.log(`  ok   ${name}`)
  } else {
    failures.push(`${name}${detail ? ' :: ' + detail : ''}`)
    console.log(`  FAIL ${name}${detail ? ' :: ' + detail : ''}`)
  }
}

const sleep = (ms) => new Promise(r => setTimeout(r, ms))

async function waitFor(fn, label, timeout = 10000) {
  const deadline = Date.now() + timeout
  let last
  while (Date.now() < deadline) {
    try {
      last = await fn()
      if (last) return last
    } catch (e) {
      last = e
    }
    await sleep(150)
  }
  throw new Error(`timeout waiting for ${label} (last: ${last})`)
}

// --- CDP client -------------------------------------------------------------
class CDP {
  constructor(ws) {
    this.ws = ws
    this.id = 0
    this.pending = new Map()
    ws.addEventListener('message', ev => {
      const msg = JSON.parse(ev.data)
      const p = this.pending.get(msg.id)
      if (!p) return
      this.pending.delete(msg.id)
      msg.error ? p.reject(new Error(JSON.stringify(msg.error))) : p.resolve(msg.result)
    })
  }

  send(method, params = {}) {
    const id = ++this.id
    return new Promise((resolve, reject) => {
      this.pending.set(id, {resolve, reject})
      this.ws.send(JSON.stringify({id, method, params}))
    })
  }

  async eval(expression) {
    const res = await this.send('Runtime.evaluate', {
      expression,
      returnByValue: true,
      awaitPromise: true,
    })
    if (res.exceptionDetails) {
      throw new Error(res.exceptionDetails.exception?.description ?? 'eval failed')
    }
    return res.result.value
  }

  click(selector) {
    return this.eval(`(() => {
      const el = document.querySelector(${JSON.stringify(selector)});
      if (!el) return false;
      el.click();
      return true;
    })()`)
  }

  clickText(tag, text) {
    return this.eval(`(() => {
      const el = [...document.querySelectorAll(${JSON.stringify(tag)})]
        .find(b => b.textContent.trim() === ${JSON.stringify(text)});
      if (!el) return false;
      el.click();
      return true;
    })()`)
  }

  fill(selector, value) {
    return this.eval(`(() => {
      const el = document.querySelector(${JSON.stringify(selector)});
      if (!el) return false;
      el.value = ${JSON.stringify(value)};
      el.dispatchEvent(new Event('input', {bubbles: true}));
      el.dispatchEvent(new Event('change', {bubbles: true}));
      return true;
    })()`)
  }

  count(selector) {
    return this.eval(`document.querySelectorAll(${JSON.stringify(selector)}).length`)
  }

  text(selector) {
    return this.eval(`(document.querySelector(${JSON.stringify(selector)})?.textContent ?? '').trim()`)
  }
}

// --- run --------------------------------------------------------------------
const exe = EXE
if (!existsSync(exe)) {
  console.error(`${exe} missing - run "wails3 build" first`)
  process.exit(1)
}

const workDir = mkdtempSync(join(tmpdir(), 'rdp-portal-e2e-'))
cpSync(exe, join(workDir, 'RDP-Portal.exe'))
const configPath = join(workDir, 'config.json')
const readConfig = () => JSON.parse(readFileSync(configPath, 'utf8'))

const app = spawn(join(workDir, 'RDP-Portal.exe'), [], {
  cwd: workDir,
  env: {...process.env, RDP_PORTAL_CDP_PORT: String(PORT)},
  stdio: 'ignore',
})

let cdp
try {
  await waitFor(async () => {
    const res = await fetch(`http://127.0.0.1:${PORT}/json/list`)
    if (!res.ok) return false
    const targets = await res.json()
    const page = targets.find(t => t.type === 'page')
    if (!page) return false
    const ws = new WebSocket(page.webSocketDebuggerUrl)
    await new Promise((resolve, reject) => {
      ws.addEventListener('open', resolve, {once: true})
      ws.addEventListener('error', reject, {once: true})
    })
    cdp = new CDP(ws)
    await cdp.send('Runtime.enable')
    return true
  }, 'devtools protocol')

  await waitFor(() => cdp.eval(`!!document.querySelector('.app')`), 'app shell')

  console.log('empty config')
  await waitFor(() => cdp.eval(`document.querySelectorAll('.profile-item').length > 0`), 'first row')
  check('one placeholder profile', (await cdp.count('.profile-item')) === 1)
  check('placeholder is selected', (await cdp.text('.profile-item.active')).includes('<New Profile>'))
  check('new profile is editable', (await cdp.eval(`!document.querySelector('#hostname').disabled`)) === true)

  console.log('new button does not stack placeholders')
  await cdp.clickText('button', 'New')
  await cdp.clickText('button', 'New')
  await cdp.clickText('button', 'New')
  await sleep(300)
  check('still one row', (await cdp.count('.profile-item')) === 1, `got ${await cdp.count('.profile-item')}`)

  console.log('save a profile')
  await waitFor(() => cdp.eval(`!!document.querySelector('.row-actions button')`), 'actions')
  await cdp.fill('#hostname', 'server1')
  await cdp.fill('.port', '3390')
  await cdp.fill('input[placeholder="Profile name"]', 'server1')
  await cdp.fill('#username', 'alice')
  await cdp.fill('#domain', 'LAB')
  await cdp.clickText('.row-actions button', 'Save')

  await waitFor(async () => {
    try {
      return readConfig().Profiles.length === 1
    } catch {
      return false
    }
  }, 'config written')

  const cfg = readConfig()
  check('profile persisted', cfg.Profiles.length === 1)
  check('full address joined from host + port', cfg.Profiles[0].Computer === 'server1:3390', cfg.Profiles[0].Computer)
  check('username persisted', cfg.Profiles[0].Username === 'alice')
  check('password stays encrypted', cfg.Profiles[0].EncryptedPassword === '')

  const rdpFile = readdirSync(join(workDir, 'rdp-files')).find(f => f.endsWith('.rdp'))
  const rdp = readFileSync(join(workDir, 'rdp-files', rdpFile), 'utf8')
  check('.rdp file written', !!rdpFile)
  check('.rdp has the address', rdp.includes('full address:s:server1:3390'))
  check('.rdp has the domain', rdp.includes('domain:s:LAB'))
  check('.rdp centres the window', /winposstr:s:0,1,-?\d+/.test(rdp))

  await sleep(400)
  check('list shows the saved name', (await cdp.text('.profile-item')).includes('server1'))
  check('Edit replaces Save', (await cdp.text('.row-actions button')) === 'Edit')

  console.log('validation')
  await cdp.clickText('button', 'New')
  await sleep(300)
  await cdp.clickText('.row-actions button', 'Save')
  const alertText = await waitFor(() => cdp.text('.alert'), 'alert')
  check('missing hostname rejected', alertText.includes('Hostname is required.'), alertText)
  check('no RuntimeError leaked', !alertText.includes('RuntimeError'), alertText)
  check('no panic leaked', !alertText.toLowerCase().includes('panic'), alertText)
  await cdp.clickText('button', 'Dismiss')
  await sleep(200)
  check('banner dismissed', (await cdp.count('.alert')) === 0)

  console.log('cancel a new profile')
  await cdp.clickText('.row-actions button', 'Cancel')
  await sleep(400)
  check('pending row dropped', (await cdp.count('.profile-item')) === 1, `got ${await cdp.count('.profile-item')}`)
  check('config untouched', readConfig().Profiles.length === 1)

  console.log('more rdp options is disabled only while editing')
  const moreBtn = `[...document.querySelectorAll('button')].find(b => b.textContent.includes('More RDP Options'))`
  const moreState = () => cdp.eval(`(() => {
    const b = ${moreBtn};
    return b ? {found: true, disabled: b.disabled} : {found: false};
  })()`)

  const viewState = await moreState()
  check('enabled when not editing', viewState.found && viewState.disabled === false, JSON.stringify(viewState))

  await cdp.clickText('.row-actions button', 'Edit')
  await sleep(300)
  const editState = await moreState()
  check('disabled while editing', editState.found && editState.disabled === true, JSON.stringify(editState))

  await cdp.clickText('.row-actions button', 'Cancel')
  await sleep(300)
  const afterCancel = await moreState()
  check('enabled again after Cancel', afterCancel.found && afterCancel.disabled === false, JSON.stringify(afterCancel))

  console.log('keep opening checkbox')
  await cdp.eval(`(() => {
    const el = document.querySelector('.keep-opening input');
    el.checked = false;
    el.dispatchEvent(new Event('change', {bubbles: true}));
    return true;
  })()`)
  await waitFor(() => readConfig().KeepOpening === false, 'KeepOpening persisted')
  check('keep opening persisted', readConfig().KeepOpening === false)

  console.log('selection does not shift the layout')
  // Add a second saved row, then compare geometry before/after changing selection.
  await cdp.clickText('button', 'New')
  await sleep(300)
  await cdp.fill('#hostname', 'px2')
  await cdp.fill('input[placeholder="Profile name"]', 'px2')
  await cdp.clickText('.row-actions button', 'Save')
  await sleep(500)

  const geometry = `(() => [...document.querySelectorAll('.profile-item')].map(li => {
    const r = li.getBoundingClientRect();
    const i = li.querySelector('.profile-icon').getBoundingClientRect();
    const n = li.querySelector('.profile-name').getBoundingClientRect();
    return [ +r.height.toFixed(2), +(i.y - r.y).toFixed(2), +(n.y - r.y).toFixed(2) ];
  }))()`

  const before = await cdp.eval(geometry)
  await cdp.clickText('.profile-item', 'server1')
  await sleep(300)
  const after = await cdp.eval(geometry)

  check('rows keep one height', new Set(before.map(g => g[0])).size === 1, JSON.stringify(before))
  check('icon offset identical across rows', new Set(before.map(g => g[1])).size === 1, JSON.stringify(before))
  check('text offset identical across rows', new Set(before.map(g => g[2])).size === 1, JSON.stringify(before))
  check('selecting does not move anything', JSON.stringify(before) === JSON.stringify(after), `${JSON.stringify(before)} vs ${JSON.stringify(after)}`)

  console.log('delete')
  const beforeDelete = readConfig().Profiles.length
  await cdp.eval(`window.confirm = () => true`)
  await cdp.clickText('button', 'Delete')
  await waitFor(() => readConfig().Profiles.length === beforeDelete - 1, 'profile removed from config')
  await sleep(400)
  check('one profile removed', readConfig().Profiles.length === beforeDelete - 1, `${beforeDelete} -> ${readConfig().Profiles.length}`)
  check('ui keeps the remaining rows', (await cdp.count('.profile-item')) === beforeDelete - 1, `${await cdp.count('.profile-item')} rows`)

  // Remove the rest to exercise the empty-state path.
  while (readConfig().Profiles.length > 0) {
    await cdp.clickText('button', 'Delete')
    await sleep(300)
  }
  await sleep(400)
  check('config has no profiles', readConfig().Profiles.length === 0)
  check('ui keeps one placeholder', (await cdp.count('.profile-item')) === 1)
  check('placeholder reselected', (await cdp.text('.profile-item')).includes('<New Profile>'))

  console.log('about dialog')
  await cdp.clickText('button', 'About')
  const about = await waitFor(() => cdp.text('.about-dialog'), 'about dialog')
  check('about shows the version', about.includes('RDP Portal v2.0.0'), about)
  check('about credits the author', about.includes('By Louis Lam'))
  check('about links the repo', about.includes('https://github.com/louislam/rdp-portal'))
  await cdp.clickText('button', 'OK')
  await sleep(200)
  check('about closes', (await cdp.count('.about-dialog')) === 0)
} catch (err) {
  failures.push(`fatal: ${err.message}`)
  console.error('fatal:', err)
} finally {
  app.kill()
  await sleep(400)
  rmSync(workDir, {recursive: true, force: true})
}

console.log(`\n${passed} passed, ${failures.length} failed`)
if (failures.length) {
  for (const f of failures) console.log(' - ' + f)
  process.exit(1)
}

