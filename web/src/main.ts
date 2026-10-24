import { mount } from 'svelte'
import 'katex/dist/katex.min.css'
import './hljs-theme.css'
import './app.css'
import App from './App.svelte'

const app = mount(App, { target: document.getElementById('app')! })
;(window as unknown as { __remoteHerdrStarted?: boolean }).__remoteHerdrStarted = true

export default app
