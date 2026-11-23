import {describe, it, expect, vi, beforeEach} from 'vitest'
import {mount, flushPromises} from '@vue/test-utils'
import App from '../App.vue'

// The Wails runtime is not available in tests; replace the generated bindings.
const bindings = vi.hoisted(() => ({
  List: vi.fn(),
  Add: vi.fn(),
  Save: vi.fn(),
  Remove: vi.fn(),
  Discard: vi.fn(),
  Connect: vi.fn(),
  Options: vi.fn(),
  KeepOpening: vi.fn(),
  SetKeepOpening: vi.fn(),
  AppVersion: vi.fn(),
}))

vi.mock('../../bindings/github.com/louislam/rdp-portal', () => ({PortalService: bindings}))

const profile = (over: Partial<any> = {}) => ({
  index: 0,
  name: 'server1',
  hostname: 'server1',
  port: '',
  username: 'alice',
  domain: '',
  password: '',
  filename: '',
  isNew: false,
  ...over,
})

beforeEach(() => {
  vi.clearAllMocks()
  bindings.AppVersion.mockResolvedValue('2.0.0')
  bindings.KeepOpening.mockResolvedValue(true)
  bindings.List.mockResolvedValue([])
})

const mountApp = async () => {
  const wrapper = mount(App)
  await flushPromises()
  return wrapper
}

describe('empty state', () => {
  it('hides profile-only controls when there are no profiles', async () => {
    const wrapper = await mountApp()
    expect(wrapper.find('.detail-pane').exists()).toBe(false)
    const buttons = wrapper.findAll('button').map(b => b.text())
    expect(buttons).toContain('New')
    expect(buttons).toContain('About')
    expect(buttons).not.toContain('Delete')
    expect(buttons).not.toContain('Connect')
    expect(buttons).not.toContain('Edit')
    expect(buttons).not.toContain('More RDP Options...')
    expect(wrapper.find('.keep-opening').exists()).toBe(false)
  })
})

describe('profile list', () => {
  it('fills the form from the selected profile, read-only until Edit', async () => {
    bindings.List.mockResolvedValue([profile({index: 0})])
    const wrapper = await mountApp()

    const host = wrapper.get('#hostname').element as HTMLInputElement
    expect(host.value).toBe('server1')
    expect(host.disabled).toBe(true)

    await wrapper.findAll('button').find(b => b.text() === 'Edit')!.trigger('click')
    expect((wrapper.get('#hostname').element as HTMLInputElement).disabled).toBe(false)
  })

  it('renders Edit (not Save) for a saved profile', async () => {
    bindings.List.mockResolvedValue([profile()])
    const wrapper = await mountApp()
    const labels = wrapper.findAll('.row-actions button').map(b => b.text())
    expect(labels).toEqual(['Edit', 'More RDP Options...'])
  })

  it('renders Save/Cancel (not Edit) for a new profile', async () => {
    bindings.List.mockResolvedValue([profile({name: '<New Profile>', hostname: '', isNew: true})])
    const wrapper = await mountApp()
    const labels = wrapper.findAll('.row-actions button').map(b => b.text())
    expect(labels).toEqual(['Save', 'Cancel', 'More RDP Options...'])
  })
})

describe('errors', () => {
  it('shows a plain message instead of RuntimeError when the hostname is missing', async () => {
    bindings.List.mockResolvedValue([profile({name: '<New Profile>', hostname: '', isNew: true})])
    bindings.Save.mockRejectedValue(new Error('RuntimeError: hostname is required'))

    const wrapper = await mountApp()
    await wrapper.findAll('.row-actions button').find(b => b.text() === 'Save')!.trigger('click')
    await flushPromises()

    const alert = wrapper.get('.alert')
    expect(alert.text()).toContain('Hostname is required.')
    expect(alert.text()).not.toContain('RuntimeError')
  })

  it('maps mstsc failures to a readable message', async () => {
    bindings.List.mockResolvedValue([profile()])
    bindings.Connect.mockRejectedValue(new Error('RuntimeError: mstsc.exe failed: exit status 1'))

    const wrapper = await mountApp()
    await wrapper.findAll('button').find(b => b.text() === 'Connect')!.trigger('click')
    await flushPromises()

    expect(wrapper.get('.alert').text()).toContain('Could not start mstsc.exe.')
  })

  it('hides the banner again on Dismiss', async () => {
    bindings.List.mockResolvedValue([profile()])
    bindings.Connect.mockRejectedValue(new Error('RuntimeError: mstsc.exe failed'))

    const wrapper = await mountApp()
    await wrapper.findAll('button').find(b => b.text() === 'Connect')!.trigger('click')
    await flushPromises()
    expect(wrapper.find('.alert').exists()).toBe(true)

    await wrapper.get('.alert-dismiss').trigger('click')
    expect(wrapper.find('.alert').exists()).toBe(false)
  })
})

describe('save flow', () => {
  it('sends hostname and port separately', async () => {
    bindings.List.mockResolvedValue([
      profile({name: '<New Profile>', hostname: '', port: '', username: '', isNew: true}),
    ])
    bindings.Save.mockResolvedValue(profile({name: 'server1'}))

    const wrapper = await mountApp()
    await wrapper.get('#hostname').setValue('server1')
    await wrapper.get('.port').setValue('3390')
    await wrapper.findAll('.row-actions button').find(b => b.text() === 'Save')!.trigger('click')
    await flushPromises()

    expect(bindings.Save).toHaveBeenCalledWith(0, '', 'server1', '3390', '', '', '')
  })

  it('cancels a new profile by discarding it', async () => {
    bindings.List.mockResolvedValue([profile({name: '<New Profile>', isNew: true})])
    bindings.Discard.mockResolvedValue(undefined)

    const wrapper = await mountApp()
    await wrapper.findAll('.row-actions button').find(b => b.text() === 'Cancel')!.trigger('click')
    await flushPromises()

    expect(bindings.Discard).toHaveBeenCalledWith(0)
  })
})

describe('delete and settings', () => {
  it('asks before deleting and calls Remove', async () => {
    bindings.List.mockResolvedValue([profile()])
    bindings.Remove.mockResolvedValue(undefined)
    vi.spyOn(window, 'confirm').mockReturnValue(true)

    const wrapper = await mountApp()
    await wrapper.findAll('button').find(b => b.text() === 'Delete')!.trigger('click')
    await flushPromises()

    expect(bindings.Remove).toHaveBeenCalledWith(0)
  })

  it('does not delete when cancelled', async () => {
    bindings.List.mockResolvedValue([profile()])
    vi.spyOn(window, 'confirm').mockReturnValue(false)

    const wrapper = await mountApp()
    await wrapper.findAll('button').find(b => b.text() === 'Delete')!.trigger('click')
    await flushPromises()

    expect(bindings.Remove).not.toHaveBeenCalled()
  })

  it('persists the keep-opening checkbox', async () => {
    bindings.List.mockResolvedValue([profile()])
    bindings.SetKeepOpening.mockResolvedValue(undefined)

    const wrapper = await mountApp()
    await wrapper.get('.keep-opening input').setValue(false)

    expect(bindings.SetKeepOpening).toHaveBeenCalledWith(false)
  })
})

describe('about dialog', () => {
  it('shows the version from the backend', async () => {
    const wrapper = await mountApp()
    await wrapper.findAll('button').find(b => b.text() === 'About')!.trigger('click')

    const about = wrapper.get('.about-dialog')
    expect(about.text()).toContain('RDP Portal v2.0.0')
    expect(about.text()).toContain('By Louis Lam')
    expect(about.text()).toContain('https://github.com/louislam/rdp-portal')
    expect(about.find('.about-ok').exists()).toBe(true)
  })
})
