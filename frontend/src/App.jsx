import { useCallback, useEffect, useMemo, useState } from 'react'
import { BrowserRouter, Link, NavLink, Navigate, Outlet, Route, Routes, useLocation, useNavigate, useOutletContext, useParams, useSearchParams } from 'react-router-dom'
import axios from 'axios'
import authWorkshop from './assets/auth-workshop.webp'
import dashboardVoices from './assets/dashboard-voices.webp'
import './App.css'

const API_BASE = (import.meta.env.VITE_API_URL || 'http://localhost:8080/api').replace(/\/$/, '')
const PUBLIC_BASE = (import.meta.env.VITE_PUBLIC_URL || window.location.origin).replace(/\/$/, '')
const api = axios.create({ baseURL: API_BASE })
const token = () => localStorage.getItem('livepoll-token')
const auth = () => ({ headers: { Authorization: `Bearer ${token()}` } })
const participantId = () => {
  let id = localStorage.getItem('livepoll-participant')
  if (!id) {
    id = crypto.randomUUID()
    localStorage.setItem('livepoll-participant', id)
  }
  return id
}
const publicLink = (id) => `${PUBLIC_BASE}/poll/${id}`
const errorMessage = (error, fallback) => error.response?.data?.error || fallback
const formatDate = (value) => new Intl.DateTimeFormat(undefined, { dateStyle: 'medium' }).format(new Date(value))

function Icon({ name }) {
  const icons = {
    overview: '▦', polls: '☷', create: '＋', live: '◉', analytics: '⌁', assistant: '✳', settings: '⚙', logout: '↪',
    menu: '☰', share: '↗', arrow: '→', close: '×', copy: '▢', search: '⌕', back: '←', sparkle: '✦',
  }
  return <span className="icon" aria-hidden="true">{icons[name] || '·'}</span>
}

function Landing() {
  const signedIn = Boolean(token())
  return <div className="landing-shell">
    <header className="landing-header"><Link to="/" className="brand"><span className="brand-mark">◉</span> LivePoll</Link><nav className="landing-nav" aria-label="Main navigation"><a href="#features">Features</a><a href="#about">About</a></nav><div className="landing-actions"><Link to={signedIn ? '/dashboard' : '/login'} className="plain-link">{signedIn ? 'Dashboard' : 'Sign in'}</Link><Link to={signedIn ? '/dashboard/create' : '/signup'} className="button primary">Create a poll <Icon name="arrow" /></Link></div></header>
    <main>
      <section className="hero landing-hero"><div className="hero-copy"><div className="eyebrow"><i /> REAL-TIME POLLING, MADE SIMPLE</div><h1>Good questions<br />make <em>great rooms.</em></h1><p>Create a poll, share one link, and watch the room decide together. Every vote appears live.</p><div className="hero-actions"><Link to={signedIn ? '/dashboard/create' : '/signup'} className="button primary">Create your first poll <Icon name="arrow" /></Link><Link to={signedIn ? '/dashboard' : '/login'} className="button secondary">{signedIn ? 'Open dashboard' : 'I have an account'}</Link></div><div className="hero-steps"><span>01 &nbsp; ASK</span><span>02 &nbsp; SHARE</span><span>03 &nbsp; DECIDE</span></div></div><div className="hero-art"><img src={dashboardVoices} alt="Illustration of a group sharing opinions and live poll results" /><div className="hero-live-card"><span><i /> LIVE RESULTS</span><b>Where should we go?</b><div><small>Beach day</small><strong><i style={{ width: '74%' }} /></strong><em>74%</em></div><div><small>City walk</small><strong><i style={{ width: '46%' }} /></strong><em>46%</em></div><div><small>Movie night</small><strong><i style={{ width: '28%' }} /></strong><em>28%</em></div><footer>Votes update as they arrive <span>↗</span></footer></div><span className="hero-art-spark hero-spark-one">✳</span><span className="hero-art-spark hero-spark-two">◉</span></div></section>
      <section className="landing-features" id="features"><div className="landing-section-heading"><span className="eyebrow"><i /> EVERYTHING IN ONE PLACE</span><h2>From first question<br />to clear decision.</h2><p>Give every person an easy way to join in, then see what the group thinks.</p></div><div className="landing-feature-grid"><article><span className="landing-feature-icon">✳</span><small>01 · CREATE</small><h3>Questions for every room</h3><p>Use multiple choice, Yes/No, ratings, ranking, image choice, open answers, and more.</p></article><article><span className="landing-feature-icon">↗</span><small>02 · SHARE</small><h3>One simple link</h3><p>Invite people to vote without making them create a LivePoll account.</p></article><article><span className="landing-feature-icon">◉</span><small>03 · SEE IT LIVE</small><h3>Results as they happen</h3><p>Watch votes update in real time and keep the conversation moving.</p></article><article><span className="landing-feature-icon">⌁</span><small>04 · UNDERSTAND</small><h3>Useful tools for hosts</h3><p>Explore themes, voting controls, audience opt-ins, discussions, and analytics.</p></article></div></section>
      <section className="landing-about" id="about"><div className="about-image"><img src={authWorkshop} alt="People working together around a table" /><span>Every voice can move the room.</span></div><div className="about-copy"><span className="eyebrow"><i /> ABOUT LIVEPOLL</span><h2>Better decisions start when everyone gets a say.</h2><p>LivePoll is a lightweight polling space for teams, friends, classrooms, and communities. Create a question, invite the group, and make the next step together with live results.</p><Link to={signedIn ? '/dashboard/create' : '/signup'} className="button primary">Bring your group together <Icon name="arrow" /></Link></div></section>
      <section className="landing-last-call"><div><span className="eyebrow"><i /> YOUR NEXT QUESTION IS READY</span><h2>Make room for every voice.</h2></div><Link to={signedIn ? '/dashboard/create' : '/signup'} className="button primary">Create a poll <Icon name="arrow" /></Link></section>
    </main>
    <footer className="landing-footer"><Link to="/" className="brand"><span className="brand-mark">◉</span> LivePoll</Link><span>Built for better conversations <span>·</span> Ask, share, decide together.</span><a href="#features">Features</a><a href="#about">About us</a></footer>
  </div>
}

function AuthPage({ signup = false }) {
  const location = useLocation()
  const navigate = useNavigate()
  const notice = !signup ? location.state?.notice : ''
  const [form, setForm] = useState({ name: '', email: '', password: '' })
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  async function submit(event) {
    event.preventDefault()
    setBusy(true)
    setError('')
    try {
      const { data } = await api.post(`/auth/${signup ? 'register' : 'login'}`, form)
      if (signup) {
        localStorage.removeItem('livepoll-token')
        localStorage.removeItem('livepoll-user')
        navigate('/login', { replace: true, state: { notice: 'Your account was created. Please sign in to continue.' } })
        return
      }
      localStorage.setItem('livepoll-token', data.token)
      localStorage.setItem('livepoll-user', JSON.stringify(data.user))
      navigate('/dashboard', { replace: true })
    } catch (requestError) {
      setError(errorMessage(requestError, 'Could not connect to LivePoll. Please try again.'))
    } finally {
      setBusy(false)
    }
  }
  return <div className={`auth-layout ${signup ? 'auth-signup' : 'auth-login'}`}>
    <section className="auth-visual" aria-label="LivePoll brings people together to share their views">
      <img src={authWorkshop} alt="A group placing colored voting cards on a table during a workshop" />
      <div className="auth-visual-shade" />
      <Link to="/" className="brand auth-visual-brand"><span className="brand-mark">◉</span> LivePoll</Link>
      <div className="auth-visual-copy"><span className="visual-kicker"><i /> MAKE EVERY VOICE COUNT</span><h2>{signup ? 'Start a conversation that brings everyone in.' : 'Good questions bring people together.'}</h2><p>Create a poll, invite your group, and see where the room lands.</p></div>
      <div className="auth-visual-footer"><span>◉ &nbsp; LIVE POLLING</span><span>Clear questions. Shared decisions.</span></div>
      <span className="auth-orbit orbit-one" /><span className="auth-orbit orbit-two" />
    </section>
    <main className="auth-main">
      <Link to="/" className="brand auth-mobile-brand"><span className="brand-mark">◉</span> LivePoll</Link>
      <section className="auth-card"><Link className="back-link" to="/">← Back home</Link><div className="eyebrow"><i /> {signup ? 'YOUR NEXT GREAT QUESTION' : 'WELCOME BACK'}</div><h1>{signup ? 'Create your account' : 'Sign in'}</h1><p className="muted">{signup ? 'Start a poll and bring your group into the conversation.' : 'Pick up where your group left off.'}</p><form onSubmit={submit}>
    {signup && <label>Your name<input autoComplete="name" minLength="2" maxLength="80" required value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} /></label>}
    <label>Email address<input type="email" autoComplete="email" required value={form.email} onChange={(event) => setForm({ ...form, email: event.target.value })} /></label>
    <label>Password<input type="password" autoComplete={signup ? 'new-password' : 'current-password'} minLength="8" required value={form.password} onChange={(event) => setForm({ ...form, password: event.target.value })} /></label>
    {notice && <div className="success-message" role="status">{notice}</div>}
    {error && <div className="alert" role="alert">{error}</div>}
    <button className="button primary full" type="submit" disabled={busy}>{busy ? 'Please wait…' : signup ? 'Create account' : 'Sign in'} <Icon name="arrow" /></button>
      </form><p className="auth-switch">{signup ? 'Already have an account?' : 'New to LivePoll?'} <Link to={signup ? '/login' : '/signup'}>{signup ? 'Sign in' : 'Create an account'}</Link></p></section><p className="auth-foot">Make every voice count.</p>
    </main>
  </div>
}

const navItems = [
  { to: '/dashboard', label: 'Overview', icon: 'overview', end: true },
  { to: '/dashboard/polls', label: 'My Polls', icon: 'polls' },
  { to: '/dashboard/create', label: 'Create Poll', icon: 'create' },
  { to: '/dashboard/live', label: 'Live Results', icon: 'live' },
  { to: '/dashboard/analytics', label: 'Analytics', icon: 'analytics' },
  { to: '/dashboard/assistant', label: 'AI Poll Assistant', icon: 'assistant' },
  { to: '/dashboard/settings', label: 'Profile & Settings', icon: 'settings' },
]

function DashboardLayout() {
  const navigate = useNavigate()
  const location = useLocation()
  const [user, setUser] = useState(() => {
    try { return JSON.parse(localStorage.getItem('livepoll-user') || 'null') } catch { return null }
  })
  const [mobileOpen, setMobileOpen] = useState(false)
  useEffect(() => {
    let alive = true
    api.get('/auth/me', auth()).then(({ data }) => {
      if (!alive) return
      setUser(data)
      localStorage.setItem('livepoll-user', JSON.stringify(data))
    }).catch((error) => {
      if (error.response?.status === 401) {
        localStorage.removeItem('livepoll-token')
        localStorage.removeItem('livepoll-user')
        navigate('/login', { replace: true })
      }
    })
    return () => { alive = false }
  }, [navigate])
  // Closing the mobile drawer is a direct response to route navigation.
  // eslint-disable-next-line react-hooks/set-state-in-effect
  useEffect(() => setMobileOpen(false), [location.pathname])
  function logout() {
    localStorage.removeItem('livepoll-token')
    localStorage.removeItem('livepoll-user')
    navigate('/', { replace: true })
  }
  return <div className="app-layout">
    <button className="mobile-menu-button" aria-label="Toggle navigation" aria-expanded={mobileOpen} onClick={() => setMobileOpen((open) => !open)}><Icon name="menu" /></button>
    {mobileOpen && <button className="sidebar-scrim" aria-label="Close navigation" onClick={() => setMobileOpen(false)} />}
    <aside className={`sidebar ${mobileOpen ? 'sidebar-open' : ''}`}>
      <Link to="/dashboard" className="brand"><span className="brand-mark">◉</span> LivePoll</Link>
      <div className="workspace-label">WORKSPACE</div>
      <nav className="side-nav" aria-label="Main navigation">{navItems.map((item) => <NavLink key={item.to} to={item.to} end={item.end} className={({ isActive }) => `nav-link ${isActive ? 'nav-active' : ''}`}><Icon name={item.icon} /><span>{item.label}</span></NavLink>)}</nav>
      <div className="sidebar-bottom"><div className="sidebar-help"><span className="help-spark">✳</span><b>Make your next poll better</b><p>Get help with questions, options, and results.</p><NavLink to="/dashboard/assistant">Ask LivePoll Assistant <Icon name="arrow" /></NavLink></div><button className="nav-link logout-link" onClick={logout}><Icon name="logout" /><span>Logout</span></button><div className="user-chip"><span className="avatar">{(user?.name || user?.email || 'LP').slice(0, 1).toUpperCase()}</span><span className="user-chip-copy"><b>{user?.name || 'LivePoll user'}</b><small>{user?.email || 'Signed in'}</small></span><button aria-label="Sign out" onClick={logout}>↪</button></div></div>
    </aside>
    <main className="dashboard-main"><div className="mobile-brand"><Link to="/dashboard" className="brand"><span className="brand-mark">◉</span> LivePoll</Link></div><Outlet context={{ user, setUser }} /></main>
  </div>
}

function PageHeading({ eyebrow, title, description, action }) {
  return <div className="page-heading"><div><div className="eyebrow"><i /> {eyebrow}</div><h1>{title}</h1>{description && <p className="muted">{description}</p>}</div>{action && <div className="heading-action">{action}</div>}</div>
}

function StatCard({ label, value, detail, icon, accent = '' }) {
  return <article className={`stat-card ${accent}`}><div className="stat-card-top"><span>{label}</span><span className="stat-icon">{icon}</span></div><b className="stat-value">{value}</b><p>{detail}</p></article>
}

function StatusBadge({ open }) { return <span className={`status-badge ${open ? 'is-open' : 'is-closed'}`}><i />{open ? 'Active' : 'Closed'}</span> }

function OverviewPage() {
  const [data, setData] = useState(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(true)
  useEffect(() => {
    api.get('/dashboard/summary', auth()).then((response) => setData(response.data)).catch((requestError) => setError(errorMessage(requestError, 'Could not load your dashboard.'))).finally(() => setBusy(false))
  }, [])
  const summary = data?.summary
  const popular = summary?.highestParticipation
  return <div className="page-content"><PageHeading eyebrow="YOUR WORKSPACE" title="Overview" description="A clear look at the polls you’ve created and the votes they’ve received." action={<Link className="button primary" to="/dashboard/create"><Icon name="create" /> Create Poll</Link>} />
    <section className="overview-visual"><img src={dashboardVoices} alt="Sculptural speech bubbles and voting bars representing a group result" /><div className="overview-visual-copy"><span className="visual-kicker"><i /> A BETTER WAY TO DECIDE</span><h2>Every opinion moves<br />the conversation forward.</h2><p>Ask the room. Watch the room respond.</p><Link to="/dashboard/create" className="visual-link">Start a new poll <Icon name="arrow" /></Link></div><span className="floating-vote vote-one">+1 voice</span><span className="floating-vote vote-two">◉ &nbsp; live</span></section>
    {error && <div className="alert" role="alert">{error} <button onClick={() => window.location.reload()}>Retry</button></div>}
    {busy ? <LoadingState label="Loading your overview…" /> : summary && <>
      <section className="stats-grid" aria-label="Poll summary">
        <StatCard label="Total polls" value={summary.totalPolls} detail="Polls you’ve created" icon="☷" />
        <StatCard label="Active polls" value={summary.activePolls} detail="Still open for votes" icon="◉" accent="stat-accent-green" />
        <StatCard label="Closed polls" value={summary.closedPolls} detail="Voting has ended" icon="✓" />
        <StatCard label="Total votes" value={summary.totalVotes} detail={`${Number(summary.averageVotesPerPoll || 0).toFixed(1)} average per poll`} icon="↗" accent="stat-accent-orange" />
      </section>
      <section className="overview-grid"><article className="content-card recent-card"><div className="card-heading"><div><h2>Recent polls</h2><p>Keep track of your latest questions.</p></div><Link to="/dashboard/polls" className="text-link">View all <Icon name="arrow" /></Link></div>
        {data.recentPolls?.length ? <div className="table-wrap"><table><thead><tr><th>Poll</th><th>Status</th><th>Votes</th><th>Created</th><th><span className="sr-only">Actions</span></th></tr></thead><tbody>{data.recentPolls.map((poll) => <tr key={poll.pollId}><td><Link className="table-question" to={`/dashboard/live/${poll.pollId}`}>{poll.question}</Link></td><td><StatusBadge open={poll.open} /></td><td>{poll.totalVotes}</td><td>{formatDate(poll.createdAt)}</td><td><Link className="icon-button" to={`/dashboard/live/${poll.pollId}`} aria-label="View live results"><Icon name="arrow" /></Link></td></tr>)}</tbody></table></div> : <EmptyState icon="✳" title="Your first poll starts here" text="Create a poll and invite your audience to weigh in." action={<Link to="/dashboard/create" className="button primary">Create a poll</Link>} />}
      </article><aside className="overview-side"><article className="content-card popular-card"><div className="card-heading"><div><h2>Most participated</h2><p>Your highest-vote poll</p></div><span className="feature-badge">✦</span></div>{popular ? <><Link to={`/dashboard/live/${popular.pollId}`} className="popular-question">{popular.question}</Link><div className="popular-total"><b>{popular.totalVotes}</b><span>votes received</span></div><div className="popular-meter"><i style={{ width: `${summary.totalVotes ? Math.max(5, popular.totalVotes / summary.totalVotes * 100) : 0}%` }} /></div></> : <p className="muted small">Once people vote, your most participated poll will appear here.</p>}</article>
        <article className="quick-actions"><h2>Quick actions</h2><Link to="/dashboard/create"><span className="quick-icon">＋</span><span><b>Create a poll</b><small>Ask the room a question</small></span><Icon name="arrow" /></Link><Link to="/dashboard/analytics"><span className="quick-icon">⌁</span><span><b>View analytics</b><small>Explore participation</small></span><Icon name="arrow" /></Link><Link to="/dashboard/polls"><span className="quick-icon">↗</span><span><b>Share a poll</b><small>Copy a public voting link</small></span><Icon name="arrow" /></Link></article>
      </aside></section>
    </>}
  </div>
}

function LoadingState({ label = 'Loading…' }) { return <div className="loading-state" role="status"><span className="spinner" />{label}</div> }
function EmptyState({ icon = '✳', title, text, action }) { return <div className="empty-state"><span className="empty-icon">{icon}</span><h3>{title}</h3><p>{text}</p>{action}</div> }

function MyPollsPage() {
  const [search, setSearch] = useState('')
  const [status, setStatus] = useState('all')
  const [sort, setSort] = useState('newest')
  const [page, setPage] = useState(1)
  const [payload, setPayload] = useState({ polls: [], total: 0, limit: 20 })
  const [busy, setBusy] = useState(true)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [searchParams] = useSearchParams()
  const pageSize = 20
  const load = useCallback(() => {
    setBusy(true)
    api.get('/polls', { ...auth(), params: { page, limit: pageSize, status, sort, q: search } }).then(({ data }) => setPayload(data)).catch((requestError) => setError(errorMessage(requestError, 'Could not load your polls.'))).finally(() => setBusy(false))
  }, [page, status, sort, search])
  // Load server data whenever the query changes.
  // eslint-disable-next-line react-hooks/set-state-in-effect
  useEffect(() => { load() }, [load])
  // eslint-disable-next-line react-hooks/set-state-in-effect
  useEffect(() => { if (searchParams.get('created')) setNotice('Your poll was created. Share the public link when you’re ready.') }, [searchParams])
  // eslint-disable-next-line react-hooks/set-state-in-effect
  useEffect(() => { setPage(1) }, [status, sort, search])
  const maxPage = Math.max(1, Math.ceil(payload.total / pageSize))
  async function copy(poll) {
    try { await navigator.clipboard.writeText(publicLink(poll.id)); setNotice('Public link copied to clipboard.') } catch { setNotice(publicLink(poll.id)) }
  }
  async function close(poll) {
    if (!window.confirm('Close voting for this poll? Participants will still be able to see its results.')) return
    try { await api.patch(`/polls/${poll.id}/close`, {}, auth()); setNotice('Poll closed.'); load() } catch (requestError) { setError(errorMessage(requestError, 'Could not close poll.')) }
  }
  async function edit(poll) {
    const question = window.prompt('Edit the poll question', poll.question)
    if (question === null) return
    try { await api.patch(`/polls/${poll.id}`, { question }, auth()); setNotice('Poll question updated.'); load() } catch (requestError) { setError(errorMessage(requestError, 'Could not update poll.')) }
  }
  async function remove(poll) {
    if (!window.confirm(`Delete “${poll.question}” and its votes? This cannot be undone.`)) return
    try { await api.delete(`/polls/${poll.id}`, auth()); setNotice('Poll deleted.'); load() } catch (requestError) { setError(errorMessage(requestError, 'Could not delete poll.')) }
  }
  return <div className="page-content"><PageHeading eyebrow="YOUR COLLECTION" title="My Polls" description="Search, sort, share, and manage the polls you own." action={<Link to="/dashboard/create" className="button primary"><Icon name="create" /> Create Poll</Link>} />
    {notice && <div className="notice" role="status">{notice}<button onClick={() => setNotice('')} aria-label="Dismiss message">×</button></div>}{error && <div className="alert" role="alert">{error}<button onClick={load}>Retry</button></div>}
    <section className="content-card poll-manager"><div className="filter-toolbar"><label className="search-box"><Icon name="search" /><input aria-label="Search polls" placeholder="Search your polls…" value={search} onChange={(event) => setSearch(event.target.value)} /></label><label className="filter-select">Status<select value={status} onChange={(event) => setStatus(event.target.value)}><option value="all">All polls</option><option value="active">Active</option><option value="closed">Closed</option></select></label><label className="filter-select">Sort by<select value={sort} onChange={(event) => setSort(event.target.value)}><option value="newest">Newest first</option><option value="oldest">Oldest first</option><option value="votes">Most votes</option></select></label></div>
      {busy ? <LoadingState label="Loading polls…" /> : payload.polls.length ? <><div className="table-wrap"><table><thead><tr><th>Poll</th><th>Status</th><th>Options</th><th>Votes</th><th>Created</th><th>Manage</th></tr></thead><tbody>{payload.polls.map((poll) => <tr key={poll.id}><td><Link className="table-question" to={`/dashboard/live/${poll.id}`}>{poll.question}</Link><small className="id-line">ID {poll.id}</small></td><td><StatusBadge open={poll.open} /></td><td>{poll.options.length} options</td><td><b>{poll.totalVotes}</b></td><td>{formatDate(poll.createdAt)}</td><td><div className="table-actions"><Link to={`/dashboard/live/${poll.id}`} title="Live results" aria-label="Live results">◉</Link><Link to={publicLink(poll.id)} target="_blank" rel="noreferrer" title="Public page" aria-label="Open public poll">↗</Link><button onClick={() => copy(poll)} title="Copy link" aria-label="Copy public link">▢</button>{poll.open && <button onClick={() => edit(poll)} title="Edit question" aria-label="Edit poll question">✎</button>}{poll.open && <button onClick={() => close(poll)} title="Close poll" aria-label="Close poll">×</button>}<button className="danger-action" onClick={() => remove(poll)} title="Delete poll" aria-label="Delete poll">⌫</button></div></td></tr>)}</tbody></table></div><div className="pagination"><span>Showing {payload.total === 0 ? 0 : (page - 1) * pageSize + 1}–{Math.min(page * pageSize, payload.total)} of {payload.total}</span><div><button disabled={page <= 1} onClick={() => setPage((value) => value - 1)}>Previous</button><b>{page} / {maxPage}</b><button disabled={page >= maxPage} onClick={() => setPage((value) => value + 1)}>Next</button></div></div></> : <EmptyState title={search || status !== 'all' ? 'No polls match those filters' : 'Your first poll starts here'} text={search || status !== 'all' ? 'Try a different search or status.' : 'Create a poll, then share its link with your audience.'} action={search || status !== 'all' ? <button className="button secondary" onClick={() => { setSearch(''); setStatus('all') }}>Reset filters</button> : <Link className="button primary" to="/dashboard/create">Create Poll</Link>} />}
    </section>
  </div>
}

function CreatePollPage() {
  const [question, setQuestion] = useState('')
  const [options, setOptions] = useState(['', ''])
  const [optionImages, setOptionImages] = useState(['', ''])
  const [questionType, setQuestionType] = useState('multiple_choice')
  const [allowMultiple, setAllowMultiple] = useState(false)
  const [ratingScale, setRatingScale] = useState(5)
  const [gridRows, setGridRows] = useState(['', ''])
  const [correctAnswer, setCorrectAnswer] = useState('')
  const [theme, setTheme] = useState('engage')
  const [themeBackground, setThemeBackground] = useState('#f2f4ec')
  const [themeAccent, setThemeAccent] = useState('#316b4d')
  const [passcodeEnabled, setPasscodeEnabled] = useState(false)
  const [passcode, setPasscode] = useState('')
  const [opensAt, setOpensAt] = useState('')
  const [endsAt, setEndsAt] = useState('')
  const [closeAfterVotes, setCloseAfterVotes] = useState('')
  const [closedMessage, setClosedMessage] = useState('')
  const [followersEnabled, setFollowersEnabled] = useState(false)
  const [followAfterResults, setFollowAfterResults] = useState(true)
  const [followerHeadline, setFollowerHeadline] = useState('Want a say in the next poll?')
  const [tab, setTab] = useState('Questions')
  const [themeTab, setThemeTab] = useState('Themes')
  const [previewOpen, setPreviewOpen] = useState(true)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const navigate = useNavigate()
  const [minimumOpenAt] = useState(() => new Date(Date.now() + 60000).toISOString().slice(0, 16))
  const freeText = ['open_ended', 'type_answer', 'word_cloud', 'brainstorm'].includes(questionType)
  const fixedOptions = ['yes_no', 'rating_scale', 'rating_grid'].includes(questionType)
  const typeOptions = [
    ['multiple_choice', 'Multiple choice', 'Choose one answer, or several'],
    ['open_ended', 'Open-ended', 'Write an answer in your own words'],
    ['yes_no', 'Yes or no', 'One tap: yes or no'],
    ['rating_scale', 'Rating scale', 'Rate it out of 5, 7, or 10'],
    ['image_choice', 'Image choice', 'Choose from image cards'],
    ['rating_grid', 'Rating grid', 'Rate several statements'],
    ['type_answer', 'Type answer', 'Type the right answer'],
    ['ranking', 'Ranking', 'Put the answers in order'],
    ['word_cloud', 'Word cloud', 'Collect short phrases'],
    ['brainstorm', 'Brainstorm', 'Share ideas with the group'],
  ]
  const themes = [
    ['engage', 'Engage', '#e7ede7', '#316b4d'], ['trust', 'Trust', '#e8f0fb', '#3169d8'], ['warmth', 'Warmth', '#f5eadb', '#b7791f'],
    ['calm', 'Calm', '#f0e9fa', '#7650c8'], ['navy', 'Navy', '#172339', '#86b7ff'], ['serendipity', 'Serendipity', '#dce8ff', '#3366d6'],
  ]
  function updateOption(index, value) { setOptions((current) => current.map((option, i) => i === index ? value : option)) }
  function updateImage(index, value) { setOptionImages((current) => current.map((item, i) => i === index ? value : item)) }
  function changeType(value) {
    setQuestionType(value)
    setAllowMultiple(false)
    if (value === 'yes_no') setOptions(['Yes', 'No'])
    else if (value === 'rating_scale' || value === 'rating_grid') setOptions(Array.from({ length: ratingScale }, (_, i) => String(i + 1)))
    else if (['open_ended', 'type_answer', 'word_cloud', 'brainstorm'].includes(value)) setOptions([])
    else if (options.length < 2) setOptions(['', ''])
  }
  function changeScale(value) {
    const scale = Number(value)
    setRatingScale(scale)
    setOptions(Array.from({ length: scale }, (_, i) => String(i + 1)))
  }
  function validate() {
    if (question.trim().length < 5 || question.trim().length > 180) return 'The question must be 5–180 characters.'
    if (!freeText && questionType !== 'rating_grid' && (options.length < 2 || (questionType !== 'rating_scale' && options.length > 8))) return 'Add a valid number of answer options.'
    if (questionType === 'image_choice' && optionImages.some((url, i) => !url.trim() || options[i]?.trim() === '')) return 'Add a label and image URL for every image choice.'
    if (questionType === 'type_answer' && !correctAnswer.trim()) return 'Add the expected answer for this quiz question.'
    if (questionType === 'rating_grid' && gridRows.filter((row) => row.trim()).length < 2) return 'Add at least two statements to the rating grid.'
    if (!freeText && !fixedOptions) {
      const normalized = options.map((option) => option.trim())
      if (normalized.some((option) => !option || option.length > 80)) return 'Each option must contain 1–80 characters.'
      if (new Set(normalized.map((option) => option.toLocaleLowerCase())).size !== normalized.length) return 'Answer options must be unique.'
    }
    if (passcodeEnabled && (passcode.length < 4 || passcode.length > 32)) return 'A passcode must be 4–32 characters.'
    if (opensAt && endsAt && new Date(endsAt) <= new Date(opensAt)) return 'The end time must be after the start time.'
    if (endsAt && new Date(endsAt) <= new Date()) return 'The end time must be in the future.'
    return ''
  }
  async function submit(event) {
    event.preventDefault()
    const message = validate()
    if (message) { setError(message); setTab('Questions'); return }
    setBusy(true); setError('')
    try {
      const localDate = opensAt ? new Date(opensAt).toISOString() : ''
      const payload = {
        question: question.trim(), questionType, options: options.map((option) => option.trim()),
        optionImages: questionType === 'image_choice' ? optionImages : [], allowMultiple,
        ratingScale: Number(ratingScale), gridRows: questionType === 'rating_grid' ? gridRows.map((row) => row.trim()).filter(Boolean) : [],
        theme, themeBackground, themeAccent, passcode: passcodeEnabled ? passcode.trim() : '', opensAt: localDate,
        endsAt: endsAt ? new Date(endsAt).toISOString() : '',
        closeAfterVotes: Number(closeAfterVotes || 0), closedMessage: closedMessage.trim(),
        followersEnabled, followAfterResults, followerHeadline: followerHeadline.trim(), correctAnswer: correctAnswer.trim(),
      }
      const { data } = await api.post('/polls', payload, auth())
      navigate(`/dashboard/live/${data.id}`)
    } catch (requestError) { setError(errorMessage(requestError, 'Could not create the poll. Please try again.')) } finally { setBusy(false) }
  }
  return <div className="page-content create-page">
    <PageHeading eyebrow="START A CONVERSATION" title="Create a poll" description="Shape the question, choose a look, and set how voting works." action={<Link to="/dashboard/polls" className="button secondary">Cancel</Link>} />
    <nav className="editor-tabs" aria-label="Poll setup sections">{['Questions', 'Theme', 'Followers', 'Settings', 'Share'].map((name) => <button type="button" key={name} className={tab === name ? 'editor-tab active' : 'editor-tab'} onClick={() => setTab(name)}>{name}</button>)}</nav>
    <div className="create-grid"><section className="content-card form-panel"><form onSubmit={submit} noValidate>
      {tab === 'Questions' && <div className="editor-section">
        <label className="field-label">Poll question <span>{question.length}/180</span><textarea required minLength="5" maxLength="180" rows="3" placeholder="Type your question here" value={question} onChange={(event) => setQuestion(event.target.value)} /></label>
        <label className="editor-type-label">Question type<select value={questionType} onChange={(event) => changeType(event.target.value)}>{typeOptions.map(([value, label]) => <option value={value} key={value}>{label}</option>)}</select></label>
        {(questionType === 'rating_scale' || questionType === 'rating_grid') && <label className="editor-setting-inline">Scale length<select value={ratingScale} onChange={(event) => changeScale(event.target.value)}><option value="5">1–5</option><option value="7">1–7</option><option value="10">1–10</option></select></label>}
        {freeText && <><div className="editor-note">{questionType === 'type_answer' ? 'Participants type an answer. You can compare it with an expected answer.' : questionType === 'word_cloud' ? 'Short responses are collected and shown as a live list.' : questionType === 'brainstorm' ? 'Invite participants to add one concise idea each.' : 'Participants submit a short text response.'}</div>{questionType === 'type_answer' && <label className="field-label">Expected answer<input required value={correctAnswer} maxLength="120" onChange={(event) => setCorrectAnswer(event.target.value)} placeholder="Answer key (kept private from voters)" /></label>}</>}
        {!freeText && questionType !== 'rating_grid' && <><div className="options-heading"><div><b>{questionType === 'ranking' ? 'Rank these answers' : questionType === 'image_choice' ? 'Image answers' : 'Answers'}</b><small>{fixedOptions ? 'These answers are generated from the selected type.' : 'Add at least 2 and no more than 8.'}</small></div><span>{options.length}</span></div><div className="create-option-list">{options.map((option, index) => <div className={`create-option ${questionType === 'image_choice' ? 'image-option-fields' : ''}`} key={index}><span className="option-number">{String(index + 1).padStart(2, '0')}</span><input aria-label={`Answer option ${index + 1}`} required={!fixedOptions} maxLength="80" disabled={fixedOptions} placeholder={`Answer ${index + 1}`} value={option} onChange={(event) => updateOption(index, event.target.value)} />{questionType === 'image_choice' && <input aria-label={`Image URL for answer ${index + 1}`} placeholder="https://… image URL" value={optionImages[index] || ''} onChange={(event) => updateImage(index, event.target.value)} />}{!fixedOptions && options.length > 2 && <button type="button" className="icon-button" aria-label={`Remove answer ${index + 1}`} onClick={() => { setOptions((current) => current.filter((_, i) => i !== index)); setOptionImages((current) => current.filter((_, i) => i !== index)) }}>×</button>}</div>)}</div>{!fixedOptions && options.length < 8 && <button type="button" className="text-button" onClick={() => { setOptions((current) => [...current, '']); setOptionImages((current) => [...current, '']) }}>＋ Add an answer</button>}{questionType === 'multiple_choice' && <label className="setting-toggle"><span><b>Allow several answers</b><small>Participants can select more than one option.</small></span><input type="checkbox" checked={allowMultiple} onChange={(event) => setAllowMultiple(event.target.checked)} /></label>}</>}
        {questionType === 'rating_grid' && <><div className="options-heading"><div><b>Statements</b><small>Participants rate each statement using the selected scale.</small></div></div><div className="create-option-list">{gridRows.map((row, index) => <div className="create-option" key={index}><span className="option-number">{String(index + 1).padStart(2, '0')}</span><input aria-label={`Grid statement ${index + 1}`} value={row} maxLength="100" placeholder={`Statement ${index + 1}`} onChange={(event) => setGridRows((current) => current.map((item, i) => i === index ? event.target.value : item))} />{gridRows.length > 2 && <button type="button" className="icon-button" onClick={() => setGridRows((current) => current.filter((_, i) => i !== index))} aria-label={`Remove statement ${index + 1}`}>×</button>}</div>)}</div>{gridRows.length < 10 && <button type="button" className="text-button" onClick={() => setGridRows((current) => [...current, ''])}>＋ Add a statement</button>}</>}
      </div>}
      {tab === 'Theme' && <div className="editor-section"><div className="section-intro"><span className="eyebrow"><i /> POLL APPEARANCE</span><h2>Set the mood</h2><p>Pick a theme for the public voting and results page.</p></div><div className="editor-subtabs">{['Themes', 'Customize', 'Your themes'].map((name) => <button type="button" key={name} className={themeTab === name ? 'active' : ''} onClick={() => setThemeTab(name)}>{name}</button>)}</div>{themeTab === 'Themes' && <div className="theme-grid">{themes.map(([id, name, background, accent]) => <button type="button" key={id} className={`theme-tile theme-${id} ${theme === id ? 'selected' : ''}`} onClick={() => { setTheme(id); setThemeBackground(background); setThemeAccent(accent) }}><span className="theme-tile-art"><i style={{ background }} /><b style={{ background: accent }} /></span><span>{name}</span>{theme === id && <strong>✓</strong>}</button>)}</div>}{themeTab === 'Customize' && <div className="theme-customizer"><label>Background color<input type="color" value={themeBackground} onChange={(event) => { setTheme('custom'); setThemeBackground(event.target.value) }} /></label><label>Accent color<input type="color" value={themeAccent} onChange={(event) => { setTheme('custom'); setThemeAccent(event.target.value) }} /></label><div className="theme-preview-chip" style={{ '--poll-bg': themeBackground, '--poll-accent': themeAccent }}>A preview of your custom theme</div></div>}{themeTab === 'Your themes' && <div className="editor-note">Your custom colors are saved with this poll. Reusable account-wide themes can be added when theme presets are stored in your profile.</div>}</div>}
      {tab === 'Followers' && <div className="editor-section"><div className="section-intro"><span className="eyebrow"><i /> AUDIENCE</span><h2>Invite people back</h2><p>Collect an optional email opt-in so you can find the poll’s audience later.</p></div><label className="setting-toggle"><span><b>Let voters follow this poll</b><small>Voters can voluntarily join this poll’s follow list.</small></span><input type="checkbox" checked={followersEnabled} onChange={(event) => setFollowersEnabled(event.target.checked)} /></label><fieldset className="follower-settings" disabled={!followersEnabled}><legend>What voters see</legend><label>Headline<input value={followerHeadline} maxLength="120" onChange={(event) => setFollowerHeadline(event.target.value)} /></label><label className="setting-toggle"><span><b>Show after results</b><small>Ask after the participant has submitted their vote.</small></span><input type="checkbox" checked={followAfterResults} onChange={(event) => setFollowAfterResults(event.target.checked)} /></label><p className="editor-note">Email addresses are stored only for the poll owner. LivePoll does not send email invitations or notifications yet.</p></fieldset></div>}
      {tab === 'Settings' && <div className="editor-section"><div className="section-intro"><span className="eyebrow"><i /> VOTING RULES</span><h2>Make it work for your group</h2><p>Choose when voting opens and how it ends.</p></div><label className="field-label">Who can vote?<select value={passcodeEnabled ? 'passcode' : 'anyone'} onChange={(event) => setPasscodeEnabled(event.target.value === 'passcode')}><option value="anyone">Anyone with the link</option><option value="passcode">Anyone with the link and passcode</option></select></label>{passcodeEnabled && <label className="field-label">Voting passcode<input type="password" autoComplete="new-password" minLength="4" maxLength="32" value={passcode} onChange={(event) => setPasscode(event.target.value)} placeholder="Choose a passcode" /></label>}<label className="field-label">Voting starts<input type="datetime-local" value={opensAt} min={minimumOpenAt} onChange={(event) => setOpensAt(event.target.value)} /></label><label className="field-label">Voting ends (optional)<input type="datetime-local" value={endsAt} min={opensAt || minimumOpenAt} onChange={(event) => setEndsAt(event.target.value)} /><small>Voting closes automatically at this time.</small></label><label className="field-label">Close after this many voters<input type="number" min="0" max="1000000" value={closeAfterVotes} onChange={(event) => setCloseAfterVotes(event.target.value)} placeholder="Leave blank to keep it open" /></label><label className="field-label">Message when voting closes<input value={closedMessage} maxLength="240" onChange={(event) => setClosedMessage(event.target.value)} placeholder="This poll is closed. Thanks for taking part!" /></label><div className="security-note"><b>Participant protection</b><span>One response per browser participant is enforced. A hidden spam field filters basic automated submissions.</span></div></div>}
      {tab === 'Share' && <div className="editor-section"><div className="section-intro"><span className="eyebrow"><i /> SHARE</span><h2>One link for the room</h2><p>Your shareable poll link is created when you publish the poll.</p></div><div className="share-setup-card"><span className="share-setup-icon">↗</span><div><b>Public voting link</b><p>Create the poll first, then use Share poll on the results page to copy its link. Anyone with the link can open the public poll; a passcode may be required if you set one.</p></div></div><div className="share-setup-card"><span className="share-setup-icon">◉</span><div><b>Live results</b><p>Votes and results update automatically while the page is open. Keep the results link handy for your group.</p></div></div></div>}
      {error && <div className="alert" role="alert">{error}</div>}
      <div className="editor-footer"><button type="button" className="button secondary" onClick={() => setPreviewOpen((value) => !value)}>{previewOpen ? 'Hide preview' : 'Preview poll'}</button><div><Link to="/dashboard/polls" className="button secondary">Cancel</Link><button type="submit" className="button primary" disabled={busy}>{busy ? 'Creating…' : 'Create poll'} <Icon name="arrow" /></button></div></div>
    </form></section>
    {previewOpen && <aside className={`content-card preview-card poll-theme-${theme}`} style={{ '--poll-bg': themeBackground, '--poll-accent': themeAccent }}><div className="preview-top"><span className="preview-label">LIVE PREVIEW · {typeOptions.find(([id]) => id === questionType)?.[1]}</span><span className="status-badge is-open"><i />Preview</span></div><div className="mini-poll-3d" aria-hidden="true"><span className="mini-orb orb-a"/><span className="mini-orb orb-b"/><span className="mini-orb orb-c"/><span className="mini-ring"/></div><p className="preview-question">{question.trim() || 'Your poll question will appear here'}</p><p className="muted small">{questionType === 'open_ended' || questionType === 'type_answer' || questionType === 'word_cloud' || questionType === 'brainstorm' ? 'Share your response in your own words.' : questionType === 'rating_grid' ? `${gridRows.filter(Boolean).length || 2} statements · ${ratingScale}-point scale` : 'Choose the answer that feels right to you.'}</p>{questionType === 'rating_grid' ? <div className="preview-options">{gridRows.filter(Boolean).slice(0, 3).map((row, index) => <div className="preview-option" key={index}>{row}</div>)}</div> : freeText ? <div className="preview-option">Type your response…</div> : <div className="preview-options">{options.slice(0, 5).map((option, index) => <div className="preview-option" key={index}>{optionImages[index] && questionType === 'image_choice' && <img className="preview-option-image" src={optionImages[index]} alt="" />}{questionType === 'ranking' && <b>{index + 1}</b>}<span className="radio-dot" />{option || `Answer ${index + 1}`}</div>)}</div>}<p className="preview-footer">Theme: {theme === 'custom' ? 'Your custom colors' : themes.find(([id]) => id === theme)?.[1]}. Your audience votes through a public link.</p></aside>}
    </div>
  </div>
}

function usePollSnapshot(id) {
  const [snapshot, setSnapshot] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [connection, setConnection] = useState('connecting')
  const [lastUpdate, setLastUpdate] = useState(null)
  useEffect(() => {
    let alive = true
    // Reset loading and connection state before starting a new poll request.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setLoading(true)
    setError('')
    setConnection('connecting')
    api.get(`/polls/${id}`).then(({ data }) => { if (alive) setSnapshot(data) }).catch((requestError) => { if (alive) setError(errorMessage(requestError, 'Could not load this poll.')) }).finally(() => { if (alive) setLoading(false) })
    const eventsUrl = `${API_BASE}/polls/${encodeURIComponent(id)}/events`
    const source = new EventSource(eventsUrl)
    source.addEventListener('snapshot', (event) => {
      if (!alive) return
      const data = JSON.parse(event.data)
      setSnapshot(data)
      setConnection('live')
      setLastUpdate(new Date())
    })
    source.addEventListener('vote', (event) => {
      if (!alive) return
      const update = JSON.parse(event.data)
      setSnapshot((current) => current ? { ...current, counts: update.counts, total: update.total, gridCounts: update.gridCounts ?? current.gridCounts, rankingScores: update.rankingScores ?? current.rankingScores, textAnswers: update.textAnswers ?? current.textAnswers, correctAnswers: update.correctAnswers ?? current.correctAnswers, poll: { ...current.poll, open: update.open ?? current.poll.open } } : current)
      setConnection('live')
      setLastUpdate(new Date())
    })
    source.onerror = () => { if (alive) setConnection('reconnecting') }
    source.onopen = () => { if (alive) setConnection('connecting') }
    return () => { alive = false; source.close() }
  }, [id])
  return { snapshot, setSnapshot, loading, error, connection, lastUpdate }
}

function ResultsPanel({ snapshot, connection, lastUpdate, share = true }) {
  const total = snapshot?.total || 0
  const poll = snapshot.poll
  const options = poll.options || []
  const type = poll.questionType || 'multiple_choice'
  const statusText = connection === 'live' ? 'Live connection' : connection === 'reconnecting' ? 'Reconnecting…' : 'Connecting…'
  const [copied, setCopied] = useState(false)
  const [shareURL, setShareURL] = useState('')
  async function copy() {
    const url = publicLink(snapshot.poll.id)
    setShareURL(url)
    try {
      await navigator.clipboard.writeText(url)
      setCopied(true)
      setTimeout(() => setCopied(false), 2500)
    } catch {
      setCopied(false)
    }
  }
  return <section className="content-card results-card"><div className="results-heading"><div><span className="eyebrow">{poll.open ? 'LIVE POLL' : poll.opensAt && new Date(poll.opensAt) > new Date() ? 'SCHEDULED' : 'POLL CLOSED'}</span><h2>{poll.question}</h2><p className="muted">{poll.open ? 'Results update as votes arrive.' : poll.opensAt && new Date(poll.opensAt) > new Date() ? `Voting opens ${new Date(poll.opensAt).toLocaleString()}.` : poll.closedMessage || 'Voting has ended. Here’s where the room landed.'}</p></div>{share && <button className="button secondary share-action" onClick={copy}><Icon name="share" />{copied ? 'Link copied' : 'Share poll'}</button>}</div>
    {shareURL && <div className="share-feedback" role="status"><label>Public poll link<input readOnly value={shareURL} onFocus={(event) => event.target.select()} aria-label="Public poll link" /></label><span>{copied ? 'Copied to clipboard.' : 'Select the link above and copy it.'}{['localhost', '127.0.0.1', '::1'].includes(window.location.hostname) && ' This localhost link only works on this computer; deploy the app to share it with others.'}</span></div>}
    <div className="results-meta"><b>{total} {total === 1 ? 'participant' : 'participants'}</b><span className={`connection-pill connection-${connection}`}><i />{statusText}</span><span>{lastUpdate ? `Updated ${lastUpdate.toLocaleTimeString()}` : 'Waiting for first update'}</span></div>
    {type === 'rating_grid' ? <div className="grid-results">{(poll.gridRows || []).map((row) => { const rowCounts = snapshot.gridCounts?.[row.id] || {}; const rowTotal = Object.values(rowCounts).reduce((sum, value) => sum + value, 0); const avg = rowTotal ? (options.reduce((sum, option) => sum + Number(option.text) * (rowCounts[option.id] || 0), 0) / rowTotal).toFixed(1) : '—'; return <div className="grid-result-row" key={row.id}><div><b>{row.text}</b><span>Average {avg}</span></div><div className="grid-result-scale">{options.map((option) => <span key={option.id} title={`${option.text}: ${rowCounts[option.id] || 0} responses`}>{rowCounts[option.id] || 0}<small>{option.text}</small></span>)}</div></div> })}</div>
      : type === 'ranking' ? <div className="result-list">{options.slice().sort((a, b) => (snapshot.rankingScores?.[b.id] || 0) - (snapshot.rankingScores?.[a.id] || 0)).map((option, index) => { const score = snapshot.rankingScores?.[option.id] || 0; const max = Math.max(1, ...Object.values(snapshot.rankingScores || {})); return <div className="result-row" key={option.id}><div className="result-label"><span>{index + 1}. {option.text}</span><b>{score} <small>points</small></b></div><div className="result-track"><i style={{ width: `${score / max * 100}%` }} /></div></div> })}</div>
        : ['open_ended', 'word_cloud', 'brainstorm'].includes(type) ? <div className={`text-answer-cloud ${type === 'word_cloud' ? 'word-cloud-mode' : ''}`}>{snapshot.textAnswers?.length ? snapshot.textAnswers.map((answer, index) => <span key={`${answer.text}-${index}`} style={type === 'word_cloud' ? { fontSize: `${Math.min(22, 12 + answer.votes * 2)}px` } : undefined}>{answer.text}<small>{answer.votes}</small></span>) : <p className="muted">Responses will appear here when people share their thoughts.</p>}</div>
          : type === 'type_answer' ? <div className="quiz-result"><b>{snapshot.correctAnswers || 0}</b><span>correct answers out of {total} responses</span></div>
            : <div className="result-list">{options.map((option) => { const count = snapshot.counts?.[option.id] || 0; const percent = total > 0 ? (count / total) * 100 : 0; return <div className="result-row" key={option.id}><div className="result-label"><span>{option.imageUrl && <img className="result-option-image" src={option.imageUrl} alt="" />}{option.text}</span><b>{count} <small>{percent.toFixed(0)}%</small></b></div><div className="result-track"><i style={{ width: `${Math.min(100, percent)}%` }} /></div></div> })}</div>}
  </section>
}

function LiveResultsPage() {
  const [polls, setPolls] = useState([])
  const [busy, setBusy] = useState(true)
  useEffect(() => { api.get('/polls', { ...auth(), params: { page: 1, limit: 100, status: 'active' } }).then(({ data }) => setPolls(data.polls || [])).catch(() => setPolls([])).finally(() => setBusy(false)) }, [])
  return <div className="page-content"><PageHeading eyebrow="LIVE RESULTS" title="Choose a poll" description="Open a poll’s live results to see new votes arrive in real time." />{busy ? <LoadingState label="Loading active polls…" /> : polls.length ? <section className="poll-card-grid">{polls.map((poll) => <Link className="live-poll-card" key={poll.id} to={`/dashboard/live/${poll.id}`}><span className="status-badge is-open"><i />Live</span><h2>{poll.question}</h2><p>{poll.totalVotes} votes <span>·</span> {poll.options.length} options</p><span className="text-link">View live results <Icon name="arrow" /></span></Link>)}</section> : <EmptyState title="No active polls" text="Create a poll to start receiving live results." action={<Link className="button primary" to="/dashboard/create">Create Poll</Link>} />}</div>
}

function PollDiscussion({ pollId }) {
  const [comments, setComments] = useState([])
  const [name, setName] = useState('')
  const [text, setText] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [sort, setSort] = useState('newest')
  const load = useCallback(() => api.get(`/polls/${pollId}/comments`).then(({ data }) => setComments(data.comments || [])).catch(() => setError('Could not load the discussion.')), [pollId])
  useEffect(() => { load() }, [load])
  async function submit(event) {
    event.preventDefault()
    if (!name.trim() || text.trim().length < 2) { setError('Add your name and a short comment.'); return }
    setBusy(true); setError('')
    try {
      await api.post(`/polls/${pollId}/comments`, { name: name.trim(), text: text.trim() })
      setText('')
      await load()
    } catch (requestError) { setError(errorMessage(requestError, 'Could not add your comment.')) } finally { setBusy(false) }
  }
  const visible = sort === 'newest' ? comments : [...comments].reverse()
  return <section className="content-card discussion-card"><div className="discussion-heading"><div><h2>Conversation</h2><p>{comments.length} {comments.length === 1 ? 'comment' : 'comments'} · Keep it thoughtful and kind.</p></div><label>Sort<select value={sort} onChange={(event) => setSort(event.target.value)}><option value="newest">Newest</option><option value="oldest">Oldest</option></select></label></div><form className="comment-form" onSubmit={submit}><label>Your name<input value={name} maxLength="60" onChange={(event) => setName(event.target.value)} placeholder="Name shown with your comment" /></label><label>Comment<textarea rows="2" maxLength="500" value={text} onChange={(event) => setText(event.target.value)} placeholder="Add to the conversation…" /></label>{error && <p className="alert" role="alert">{error}</p>}<button className="button secondary" type="submit" disabled={busy}>{busy ? 'Posting…' : 'Post comment'} <Icon name="arrow" /></button></form><div className="comment-list">{visible.map((comment) => <article className="comment-item" key={comment.id}><span className="comment-avatar">{comment.name.slice(0, 1).toUpperCase()}</span><div><p><b>{comment.name}</b><time>{new Date(comment.createdAt).toLocaleString()}</time></p><span>{comment.text}</span></div></article>)}{!visible.length && <p className="muted">Be the first to share a thought.</p>}</div></section>
}

function FollowerSignup({ poll }) {
  const [email, setEmail] = useState('')
  const [firstName, setFirstName] = useState('')
  const [message, setMessage] = useState('')
  const [busy, setBusy] = useState(false)
  async function submit(event) {
    event.preventDefault(); setBusy(true); setMessage('')
    try { const { data } = await api.post(`/polls/${poll.id}/followers`, { email, firstName }); setMessage(data.message); setEmail(''); setFirstName('') }
    catch (error) { setMessage(errorMessage(error, 'Could not save your follow request.')) } finally { setBusy(false) }
  }
  return <form className="follower-signup" onSubmit={submit}><h3>{poll.followerHeadline || 'Want a say in the next poll?'}</h3><p>Opt in to the poll owner’s audience list. Your email is visible only to the owner; no email is sent by LivePoll.</p><label>Email<input type="email" required autoComplete="email" value={email} onChange={(event) => setEmail(event.target.value)} /></label><label>First name <small>(optional)</small><input autoComplete="given-name" maxLength="80" value={firstName} onChange={(event) => setFirstName(event.target.value)} /></label><button className="button primary" disabled={busy}>{busy ? 'Saving…' : 'Follow this poll'}</button>{message && <p className="muted" role="status">{message}</p>}</form>
}

function PollParticipants({ pollId }) {
  const [participants, setParticipants] = useState([])
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)
  useEffect(() => {
    let active = true
    api.get(`/polls/${pollId}/participants`, auth())
      .then(({ data }) => { if (active) setParticipants(data.participants || []) })
      .catch((requestError) => { if (active) setError(errorMessage(requestError, 'Could not load participation.')) })
      .finally(() => { if (active) setLoading(false) })
    return () => { active = false }
  }, [pollId])
  return <section className="content-card participants-card"><div className="card-heading"><div><h2>Participation</h2><p>Responses and names shared by participants. Names are optional.</p></div><span className="participants-count">{participants.length}</span></div>{error ? <p className="alert" role="alert">{error}</p> : loading ? <p className="muted">Loading participation…</p> : participants.length ? <div className="participants-list">{participants.map((participant, index) => <article className="participant-row" key={`${participant.createdAt}-${index}`}><span className="comment-avatar">{participant.name.slice(0, 1).toUpperCase()}</span><div><b>{participant.name}</b><p>{participant.response || 'Response recorded'}</p></div><time>{new Date(participant.createdAt).toLocaleString()}</time></article>)}</div> : <p className="muted">No one has voted yet. New responses will appear here.</p>}</section>
}

function PollFollowersManager({ pollId }) {
  const [followers, setFollowers] = useState([])
  const [error, setError] = useState('')
  useEffect(() => { api.get(`/polls/${pollId}/followers`, auth()).then(({ data }) => setFollowers(data.followers || [])).catch((requestError) => setError(errorMessage(requestError, 'Could not load the follower list.'))) }, [pollId])
  function exportCSV() {
    const csv = [['Email', 'First name', 'Joined'], ...followers.map((item) => [item.email, item.firstName || '', item.createdAt])].map((row) => row.map((value) => `"${String(value).replaceAll('"', '""')}"`).join(',')).join('\r\n')
    const url = URL.createObjectURL(new Blob([csv], { type: 'text/csv;charset=utf-8' })); const link = document.createElement('a'); link.href = url; link.download = 'livepoll-followers.csv'; link.click(); URL.revokeObjectURL(url)
  }
  return <section className="content-card followers-manager"><div className="card-heading"><div><h2>Followers</h2><p>People who opted in to this poll’s audience list.</p></div><button className="button secondary" onClick={exportCSV} disabled={!followers.length}>Export CSV</button></div>{error ? <p className="alert" role="alert">{error}</p> : followers.length ? <div className="follower-list">{followers.map((follower) => <div key={follower.id}><span className="comment-avatar">{(follower.firstName || follower.email).slice(0, 1).toUpperCase()}</span><b>{follower.firstName || 'Poll follower'}</b><span>{follower.email}</span><time>{formatDate(follower.createdAt)}</time></div>)}</div> : <p className="muted">No one has opted in yet.</p>}</section>
}

function PollPage({ managed = false }) {
  const { id } = useParams()
  const { snapshot, setSnapshot, loading, error, connection, lastUpdate } = usePollSnapshot(id)
  const [selected, setSelected] = useState('')
  const [selectedMany, setSelectedMany] = useState([])
  const [answerText, setAnswerText] = useState('')
  const [grid, setGrid] = useState({})
  const [ranking, setRanking] = useState([])
  const [passcode, setPasscode] = useState('')
  const [voterName, setVoterName] = useState('')
  const [voted, setVoted] = useState(false)
  const [busy, setBusy] = useState(false)
  const [message, setMessage] = useState('')
  function moveRank(index, direction, optionIds) {
    const current = ranking.length ? [...ranking] : optionIds
    const next = index + direction
    if (next < 0 || next >= current.length) return
    ;[current[index], current[next]] = [current[next], current[index]]
    setRanking(current)
  }
  async function vote() {
    if (voted || busy) return
    const poll = snapshot.poll
    const type = poll.questionType || 'multiple_choice'
    const hasResponse = ['open_ended', 'word_cloud', 'brainstorm', 'type_answer'].includes(type) ? answerText.trim() : type === 'rating_grid' ? poll.gridRows?.every((row) => grid[row.id]) : type === 'ranking' ? (ranking.length || poll.options.length) : (poll.allowMultiple ? selectedMany.length : selected)
    if (!hasResponse) { setMessage('Choose or enter your response before submitting.'); return }
    setBusy(true); setMessage('')
    const payload = { voterId: participantId(), voterName: voterName.trim(), passcode, website: '' }
    if (type === 'rating_grid') payload.grid = grid
    else if (type === 'ranking') payload.ranking = ranking.length ? ranking : poll.options.map((option) => option.id)
    else if (['open_ended', 'word_cloud', 'brainstorm', 'type_answer'].includes(type)) payload.answerText = answerText
    else if (poll.allowMultiple) payload.optionIds = selectedMany
    else payload.optionId = selected
    try {
      const { data } = await api.post(`/polls/${id}/votes`, payload)
      setSnapshot((current) => current ? { ...current, counts: data.counts || current.counts, total: data.total ?? current.total, gridCounts: data.gridCounts ?? current.gridCounts, rankingScores: data.rankingScores ?? current.rankingScores, textAnswers: data.textAnswers ?? current.textAnswers, correctAnswers: data.correctAnswers ?? current.correctAnswers, poll: { ...current.poll, open: data.open ?? current.poll.open } } : current)
      setVoted(true); setMessage(type === 'type_answer' ? 'Your answer is in. Check the results to see if it matched.' : 'Your vote is in. Thanks for weighing in!')
    } catch (requestError) { setMessage(errorMessage(requestError, 'Could not submit your vote. Please try again.')) } finally { setBusy(false) }
  }
  if (loading) return <div className={managed ? 'page-content' : 'public-poll-shell'}><LoadingState label="Loading poll…" /></div>
  if (error || !snapshot) return managed ? <div className="page-content"><EmptyState title="Poll unavailable" text={error || 'This poll could not be found.'} action={<Link to="/dashboard/polls" className="button secondary">Back to My Polls</Link>} /></div> : <div className="public-poll-shell"><EmptyState title="We can’t find that poll" text={error || 'This poll does not exist.'} action={<Link to="/" className="button secondary">Back home</Link>} /></div>
  const poll = snapshot.poll
  const type = poll.questionType || 'multiple_choice'
  const localScheduled = Boolean(poll.opensAt && new Date(poll.opensAt) > new Date())
  const wrapperStyle = { '--poll-bg': poll.themeBackground || '#f2f4ec', '--poll-accent': poll.themeAccent || '#316b4d' }
  const showFollowers = !managed && poll.followersEnabled && (poll.followAfterResults ? voted : true)
  return <div className={`${managed ? 'page-content' : `public-poll-shell poll-theme-${poll.theme || 'engage'}`}`} style={!managed ? wrapperStyle : undefined}>
    {managed ? <PageHeading eyebrow="LIVE RESULTS" title="Poll results" description="Vote totals update live for everyone viewing this poll." action={<Link className="button secondary" to="/dashboard/polls"><Icon name="back" /> My Polls</Link>} /> : <div className="public-poll-brand"><Link to="/" className="brand"><span className="brand-mark">◉</span> LivePoll</Link><span>Make every voice count.</span></div>}
    <ResultsPanel snapshot={snapshot} connection={connection} lastUpdate={lastUpdate} />
    {managed && <PollParticipants pollId={id} />}
    {managed && poll.followersEnabled && <PollFollowersManager pollId={id} />}
    {!managed && <section className="content-card vote-card"><div className="eyebrow">{poll.open ? 'CAST YOUR VOTE' : localScheduled ? 'VOTING OPENS SOON' : 'VOTING HAS ENDED'}</div>{poll.open ? <><h2>{type === 'rating_grid' ? 'Rate each statement' : type === 'ranking' ? 'Put these answers in order' : type === 'type_answer' ? 'Type your answer' : 'What do you think?'}</h2>
      <label className="field-label voter-name-field">Your name (optional)<input value={voterName} maxLength="60" disabled={voted} onChange={(event) => setVoterName(event.target.value)} placeholder="Shown to the poll owner" /></label>
      {['open_ended', 'word_cloud', 'brainstorm', 'type_answer'].includes(type) ? <textarea className="answer-text-input" rows="3" maxLength="500" value={answerText} disabled={voted} onChange={(event) => setAnswerText(event.target.value)} placeholder={type === 'brainstorm' ? 'Share one clear idea…' : type === 'word_cloud' ? 'Add a word or short phrase…' : 'Type your response…'} />
        : type === 'rating_grid' ? <div className="rating-grid-vote">{(poll.gridRows || []).map((row) => <fieldset key={row.id}><legend>{row.text}</legend>{poll.options.map((option) => <label key={option.id} className={`grid-rate-choice ${grid[row.id] === option.id ? 'choice-selected' : ''}`}><input type="radio" name={`grid-${row.id}`} checked={grid[row.id] === option.id} disabled={voted} onChange={() => setGrid((current) => ({ ...current, [row.id]: option.id }))} />{option.text}</label>)}</fieldset>)}</div>
          : type === 'ranking' ? <div className="ranking-vote-list">{poll.options.map((option, index) => <div className="ranking-vote-row" key={option.id}><span>{index + 1}</span><b>{option.text}</b><button type="button" disabled={voted || index === 0} aria-label={`Move ${option.text} up`} onClick={() => moveRank(index, -1, poll.options.map((item) => item.id))}>↑</button><button type="button" disabled={voted || index === poll.options.length - 1} aria-label={`Move ${option.text} down`} onClick={() => moveRank(index, 1, poll.options.map((item) => item.id))}>↓</button></div>)}</div>
            : <div className={`vote-options ${type === 'image_choice' ? 'vote-image-grid' : ''}`}>{poll.options.map((option) => { const selectedOption = poll.allowMultiple ? selectedMany.includes(option.id) : selected === option.id; return <label className={`vote-choice ${selectedOption ? 'choice-selected' : ''}`} key={option.id}><input type={poll.allowMultiple ? 'checkbox' : 'radio'} name="poll-option" value={option.id} checked={selectedOption} disabled={voted} onChange={() => poll.allowMultiple ? setSelectedMany((current) => current.includes(option.id) ? current.filter((idValue) => idValue !== option.id) : [...current, option.id]) : setSelected(option.id)} />{option.imageUrl && <img className="vote-option-image" src={option.imageUrl} alt="" />}<span className="radio-dot" />{option.text}</label> })}</div>}
      {poll.passcodeRequired && <label className="field-label poll-passcode">Voting passcode<input type="password" autoComplete="off" value={passcode} onChange={(event) => setPasscode(event.target.value)} placeholder="Enter the poll passcode" /></label>}
      <input className="poll-honeypot" tabIndex="-1" autoComplete="off" aria-hidden="true" value="" readOnly />
      <button className="button primary" disabled={busy || voted || !poll.open} onClick={vote}>{busy ? 'Submitting…' : voted ? 'Response recorded' : 'Submit response'} <Icon name="arrow" /></button>
      </> : <p className="muted">{localScheduled ? `Voting opens ${new Date(poll.opensAt).toLocaleString()}.` : poll.closedMessage || 'This poll is closed. You can still view the results above.'}</p>}{message && <p className={message.startsWith('Your ') ? 'success-message' : 'alert'} role="status">{message}</p>}</section>}
    {showFollowers && <section className="content-card social-card"><FollowerSignup poll={poll} /></section>}
    <PollDiscussion pollId={id} />
  </div>
}

function MiniBar({ value, max, label, color = 'orange' }) {
  const width = max > 0 ? Math.max(value > 0 ? 2 : 0, value / max * 100) : 0
  return <div className="mini-bar-row"><span title={label}>{label}</span><div className="mini-bar-track"><i className={`mini-bar-${color}`} style={{ width: `${width}%` }} /></div><b>{value}</b></div>
}

function LineChart({ data }) {
  if (!data.length) return <EmptyState title="No votes in this range" text="Votes will appear here once participants submit responses." />
  const width = 720; const height = 220; const pad = 28
  const max = Math.max(1, ...data.map((item) => item.votes))
  const points = data.map((item, index) => ({ x: pad + (data.length === 1 ? (width - 2 * pad) / 2 : index * (width - 2 * pad) / (data.length - 1)), y: height - pad - item.votes / max * (height - 2 * pad), ...item }))
  return <div className="line-chart-wrap"><svg className="line-chart" viewBox={`0 0 ${width} ${height}`} role="img" aria-label="Votes per day over the selected date range"><line x1={pad} y1={height - pad} x2={width - pad} y2={height - pad} className="chart-axis" />{[0, 0.5, 1].map((step) => <line key={step} x1={pad} y1={height - pad - step * (height - 2 * pad)} x2={width - pad} y2={height - pad - step * (height - 2 * pad)} className="chart-gridline" />)}<polyline points={points.map((point) => `${point.x},${point.y}`).join(' ')} className="chart-line" />{points.map((point) => <g key={point.date}><circle cx={point.x} cy={point.y} r="4" className="chart-point"><title>{point.date}: {point.votes} votes</title></circle></g>)}</svg><div className="chart-dates"><span>{data[0].date}</span><span>{data[data.length - 1].date}</span></div><div className="chart-data-table"><span className="sr-only">{data.map((item) => `${item.date}: ${item.votes} votes`).join('; ')}</span></div></div>
}

function AnalyticsPage() {
  const [filters, setFilters] = useState({ from: '', to: '', pollId: '', status: 'all' })
  const [data, setData] = useState(null)
  const [polls, setPolls] = useState([])
  const [busy, setBusy] = useState(true)
  const [error, setError] = useState('')
  useEffect(() => { api.get('/polls', { ...auth(), params: { page: 1, limit: 100 } }).then(({ data: result }) => setPolls(result.polls || [])).catch(() => {}) }, [])
  const query = useMemo(() => Object.fromEntries(Object.entries(filters).filter(([, value]) => value)), [filters])
  const load = useCallback(() => {
    setBusy(true); setError('')
    api.get('/analytics', { ...auth(), params: query }).then(({ data: result }) => setData(result)).catch((requestError) => setError(errorMessage(requestError, 'Could not load analytics.'))).finally(() => setBusy(false))
  }, [query])
  // Recalculate the server analytics whenever filters change.
  // eslint-disable-next-line react-hooks/set-state-in-effect
  useEffect(() => { load() }, [load])
  function exportCSV() {
    if (!data) return
    const rows = [['Type', 'Date or poll', 'Question or answer', 'Votes', 'Percent']]
    for (const poll of data.votesByPoll) rows.push(['Poll', formatDate(poll.createdAt), poll.question, String(poll.totalVotes), ''])
    for (const day of data.votesOverTime) rows.push(['Day', day.date, 'Votes', String(day.votes), ''])
    for (const option of data.optionDistribution) rows.push(['Answer', '', option.text, String(option.votes), `${option.percent.toFixed(1)}%`])
    const csv = rows.map((row) => row.map((cell) => `"${String(cell).replaceAll('"', '""')}"`).join(',')).join('\r\n')
    const url = URL.createObjectURL(new Blob([csv], { type: 'text/csv;charset=utf-8' }))
    const anchor = document.createElement('a'); anchor.href = url; anchor.download = 'livepoll-analytics.csv'; anchor.click(); URL.revokeObjectURL(url)
  }
  function resetFilters() { setFilters({ from: '', to: '', pollId: '', status: 'all' }) }
  return <div className="page-content"><PageHeading eyebrow="UNDERSTAND PARTICIPATION" title="Analytics" description="Explore real voting activity from your polls." action={<button className="button secondary" onClick={exportCSV} disabled={!data}>↓ Export CSV</button>} />
    <section className="content-card analytics-filters"><label>From<input type="date" value={filters.from} onChange={(event) => setFilters({ ...filters, from: event.target.value })} /></label><label>To<input type="date" value={filters.to} onChange={(event) => setFilters({ ...filters, to: event.target.value })} /></label><label>Poll<select value={filters.pollId} onChange={(event) => setFilters({ ...filters, pollId: event.target.value })}><option value="">All my polls</option>{polls.map((poll) => <option key={poll.id} value={poll.id}>{poll.question}</option>)}</select></label><label>Status<select value={filters.status} onChange={(event) => setFilters({ ...filters, status: event.target.value })}><option value="all">All</option><option value="active">Active</option><option value="closed">Closed</option></select></label><button className="text-button filter-reset" onClick={resetFilters}>Reset filters</button></section>
    {error && <div className="alert" role="alert">{error}<button onClick={load}>Retry</button></div>}{busy ? <LoadingState label="Calculating analytics from your saved votes…" /> : data && <>
      <section className="stats-grid analytics-stats"><StatCard label="Total polls" value={data.totalPolls} detail="In selected scope" icon="☷" /><StatCard label="Total votes" value={data.totalVotes} detail={`${data.averageVotesPerPoll.toFixed(1)} average per poll`} icon="↗" accent="stat-accent-orange" /><StatCard label="Active polls" value={data.activePolls} detail={`${data.closedPolls} closed`} icon="◉" accent="stat-accent-green" /><StatCard label="Top answer" value={data.mostSelectedOption?.votes ?? '—'} detail={data.mostSelectedOption?.text || 'No votes yet'} icon="✦" /></section>
      <section className="analytics-grid">
        <article className="content-card chart-card"><div className="card-heading"><div><h2>Votes over time</h2><p>Daily vote totals, grouped in UTC.</p></div></div><LineChart data={data.votesOverTime} /></article>
        <article className="content-card chart-card"><div className="card-heading"><div><h2>Votes by poll</h2><p>Compare participation across your polls.</p></div></div>{data.votesByPoll.length ? <div className="mini-bars">{data.votesByPoll.slice(0, 8).map((poll) => <Link to={`/dashboard/live/${poll.pollId}`} key={poll.pollId}><MiniBar label={poll.question} value={poll.totalVotes} max={Math.max(1, ...data.votesByPoll.map((item) => item.totalVotes))} /></Link>)}</div> : <EmptyState title="No polls in this view" text="Adjust filters or create a poll." />}</article>
        <article className="content-card chart-card"><div className="card-heading"><div><h2>Poll status</h2><p>Open polls and polls that have closed.</p></div></div><div className="status-summary"><div><span className="status-dot active-dot" />Active<b>{data.activePolls}</b></div><div><span className="status-dot closed-dot" />Closed<b>{data.closedPolls}</b></div></div><div className="status-stacked"><i style={{ width: `${data.totalPolls ? data.activePolls / data.totalPolls * 100 : 0}%` }} /><span style={{ width: `${data.totalPolls ? data.closedPolls / data.totalPolls * 100 : 0}%` }} /></div><p className="chart-caption">{data.totalPolls ? `${data.totalPolls} polls in the selected scope` : 'No poll data yet'}</p></article>
        <article className="content-card chart-card">
          <div className="card-heading"><div><h2>Answer distribution</h2><p>Choose a poll to compare its answer options.</p></div></div>
          <select className="chart-select" aria-label="Select poll for answer distribution" value={filters.pollId} onChange={(event) => setFilters({ ...filters, pollId: event.target.value })}><option value="">Select a poll</option>{polls.map((poll) => <option key={poll.id} value={poll.id}>{poll.question}</option>)}</select>
          {filters.pollId ? (data.optionDistribution.length ? <div className="mini-bars">{data.optionDistribution.map((option) => <MiniBar key={option.optionId} label={option.text} value={option.votes} max={Math.max(1, ...data.optionDistribution.map((item) => item.votes))} />)}</div> : <EmptyState title="No votes yet" text="Answer distribution will appear after the first vote." />) : <p className="chart-caption">Select one of your polls to view its answer breakdown.</p>}
        </article>
      </section>
      <section className="content-card activity-card"><div className="card-heading"><div><h2>Recent voting activity</h2><p>Latest saved responses from your polls.</p></div></div>{data.recentActivity.length ? <div className="activity-list">{data.recentActivity.map((activity, index) => <div className="activity-row" key={`${activity.pollId}-${activity.createdAt}-${index}`}><span className="activity-dot">↗</span><p><b>{activity.option}</b> received a vote in <Link to={`/dashboard/live/${activity.pollId}`}>{activity.question}</Link></p><time>{new Date(activity.createdAt).toLocaleString()}</time></div>)}</div> : <p className="muted">No votes have been recorded in this range yet.</p>}</section>
    </>}
  </div>
}

const suggestedPrompts = ['Help me create a poll for customer feedback.', 'Suggest five questions for a college event.', 'How can I improve participation?', 'Summarize my recent poll results.', 'Help me write a neutral poll question.', 'How do I share a poll?']
function AssistantPage() {
  const [messages, setMessages] = useState([{ role: 'assistant', text: 'Hi! What would you like to work on today? I can help with poll ideas, question wording, participation, sharing, or your results.' }])
  const [input, setInput] = useState('')
  const [busy, setBusy] = useState(false)
  const [mode, setMode] = useState('guidance')
  const [copiedPollId, setCopiedPollId] = useState('')
  const [error, setError] = useState('')
  const [lastFailed, setLastFailed] = useState('')
  async function send(text = input) {
    const message = text.trim()
    if (!message || busy) return
    setMessages((current) => [...current, { role: 'user', text: message }])
    setInput(''); setBusy(true); setError(''); setLastFailed('')
    try { const history = messages.slice(-11); const { data } = await api.post('/assistant/chat', { message, messages: history }, auth()); setMode(data.mode || 'guidance'); setMessages((current) => [...current, { role: 'assistant', text: data.reply, label: data.label, pollId: data.pollId }]) }
    catch (requestError) { setError(errorMessage(requestError, 'Could not get a response. Please retry.')); setLastFailed(message) }
    finally { setBusy(false) }
  }
  async function copyCreatedPoll(pollId) {
    try { await navigator.clipboard.writeText(publicLink(pollId)); setCopiedPollId(pollId); setTimeout(() => setCopiedPollId(''), 2200) }
    catch { setError('Clipboard access failed. Open the poll and copy its address from the browser.') }
  }
  function clear() { setMessages([{ role: 'assistant', text: 'Hi again! What would you like to work on now?' }]); setError(''); setInput('') }
  return <div className="page-content assistant-page"><PageHeading eyebrow="POLL WRITING & INSIGHTS" title="LivePoll Assistant" description="Get practical help with questions, options, sharing, and your real poll metrics." action={<button className="button secondary" onClick={clear}>Clear conversation</button>} />
    <section className="assistant-shell content-card"><div className="assistant-top"><div className="assistant-avatar">✳</div><div><b>LivePoll Assistant</b><small><i /> {mode === 'ai' ? 'AI connected · Account totals are included for result questions' : 'Built-in guidance · Add a server API key to enable AI chat'}</small></div><span className="mode-tag">{mode === 'ai' ? 'AI CHAT' : 'GUIDANCE MODE'}</span></div><div className="chat-messages" aria-live="polite">{messages.map((message, index) => <article className={`chat-message message-${message.role}`} key={`${message.role}-${index}`}><div className="message-avatar">{message.role === 'assistant' ? '✳' : 'You'}</div><div className="message-copy"><p>{message.text}</p>{message.pollId && <div className="assistant-poll-actions"><Link to={`/poll/${message.pollId}`} target="_blank" rel="noreferrer">Open poll <Icon name="arrow" /></Link><button type="button" onClick={() => copyCreatedPoll(message.pollId)}>{copiedPollId === message.pollId ? 'Link copied' : 'Copy poll link'}</button></div>}{message.label && <small>{message.label}</small>}</div></article>)}{busy && <div className="chat-message message-assistant"><div className="message-avatar">✳</div><div className="typing-dots" role="status" aria-label="Assistant is preparing a response"><i /><i /><i /></div></div>}</div>{messages.length < 3 && <div className="prompt-list"><span>TRY ASKING</span><div>{suggestedPrompts.map((prompt) => <button key={prompt} onClick={() => send(prompt)} disabled={busy}>{prompt}</button>)}</div></div>}{error && <div className="alert" role="alert">{error} <button onClick={() => send(lastFailed)}>Retry</button></div>}<form className="chat-composer" onSubmit={(event) => { event.preventDefault(); send() }}><textarea aria-label="Message LivePoll Assistant" maxLength="1000" rows="2" placeholder="Ask about poll ideas, participation, or your results…" value={input} onChange={(event) => setInput(event.target.value)} /><button className="button primary" type="submit" disabled={!input.trim() || busy}>Send <Icon name="arrow" /></button></form><p className="assistant-disclaimer">{mode === 'ai' ? 'AI responses use your recent messages and account-level poll aggregates. No voter emails or passwords are sent.' : 'Built-in guidance replies to common LivePoll questions. Configure OPENAI_API_KEY on the backend for conversational AI.'}</p></section>
  </div>
}

function SettingsPage() {
  const { user, setUser } = useOutletContextSafe()
  const [busy, setBusy] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [profileError, setProfileError] = useState('')
  const [profileMessage, setProfileMessage] = useState('')
  const [profile, setProfile] = useState({ name: '', email: '' })
  const [dark, setDark] = useState(() => localStorage.getItem('livepoll-theme') === 'dark')
  useEffect(() => { api.get('/auth/me', auth()).then(({ data }) => { setUser(data); setProfile({ name: data.name || '', email: data.email || '' }); localStorage.setItem('livepoll-user', JSON.stringify(data)) }).catch((requestError) => setError(errorMessage(requestError, 'Could not load profile.'))).finally(() => setBusy(false)) }, [setUser])
  async function saveProfile(event) {
    event.preventDefault(); setSaving(true); setProfileError(''); setProfileMessage('')
    try {
      const { data } = await api.patch('/auth/me', { name: profile.name.trim(), email: profile.email.trim() }, auth())
      const updated = { ...user, ...data }
      setUser(updated); setProfile({ name: data.name, email: data.email }); localStorage.setItem('livepoll-user', JSON.stringify(updated)); setProfileMessage('Your profile was saved.')
    } catch (requestError) { setProfileError(errorMessage(requestError, 'Could not save your profile. Please try again.')) }
    finally { setSaving(false) }
  }
  function toggleTheme(event) { const next = event.target.checked; setDark(next); localStorage.setItem('livepoll-theme', next ? 'dark' : 'light'); document.documentElement.dataset.theme = next ? 'dark' : 'light' }
  useEffect(() => { document.documentElement.dataset.theme = dark ? 'dark' : 'light' }, [dark])
  return <div className="page-content"><PageHeading eyebrow="ACCOUNT" title="Profile & Settings" description="Update your account details and local preferences." />{busy ? <LoadingState label="Loading profile…" /> : <div className="settings-grid">{error && <div className="alert" role="alert">{error}</div>}<section className="content-card profile-edit-card"><div className="profile-edit-heading"><div className="profile-large-avatar">{(profile.name || profile.email || 'LP').slice(0, 1).toUpperCase()}</div><div><h2>Personal information</h2><p>Edit the name and email shown on your account.</p>{user?.createdAt && <small>Member since {formatDate(user.createdAt)}</small>}</div></div><form className="profile-form" onSubmit={saveProfile}><label>Full name<input required minLength="2" maxLength="80" autoComplete="name" value={profile.name} onChange={(event) => setProfile((current) => ({ ...current, name: event.target.value }))} /></label><label>Email address<input required type="email" maxLength="254" autoComplete="email" value={profile.email} onChange={(event) => setProfile((current) => ({ ...current, email: event.target.value }))} /></label>{profileError && <div className="alert" role="alert">{profileError}</div>}{profileMessage && <p className="profile-success" role="status">{profileMessage}</p>}<button className="button primary" type="submit" disabled={saving}>{saving ? 'Saving…' : 'Save changes'} <Icon name="arrow" /></button></form></section><section className="content-card settings-card"><div className="card-heading"><div><h2>Preferences</h2><p>These settings are saved in this browser.</p></div></div><label className="setting-row"><span><b>Dark appearance</b><small>Use a darker color palette on this device.</small></span><input type="checkbox" checked={dark} onChange={toggleTheme} /></label></section><section className="content-card settings-card"><div className="card-heading"><div><h2>Assistant configuration</h2><p>Optional AI setup is kept on the backend; API keys never enter the browser.</p></div></div><div className="setting-mode"><span className="status-dot active-dot" /><div><b>Conversational assistant</b><small>Set OPENAI_API_KEY in backend/.env to enable contextual AI replies. Without it, LivePoll uses built-in guidance.</small></div></div></section></div>}</div>
}

// useOutletContext is kept behind a tiny wrapper to make settings resilient to direct route loads.
function useOutletContextSafe() { return useOutletContext() }

export default function App() {
  return <BrowserRouter><Routes><Route path="/" element={<Landing />} /><Route path="/signup" element={<AuthPage signup />} /><Route path="/login" element={<AuthPage />} /><Route path="/dashboard" element={token() ? <DashboardLayout /> : <Navigate to="/login" replace />}><Route index element={<OverviewPage />} /><Route path="polls" element={<MyPollsPage />} /><Route path="create" element={<CreatePollPage />} /><Route path="live" element={<LiveResultsPage />} /><Route path="live/:id" element={<PollPage managed />} /><Route path="analytics" element={<AnalyticsPage />} /><Route path="assistant" element={<AssistantPage />} /><Route path="settings" element={<SettingsPage />} /></Route><Route path="/poll/:id" element={<PollPage />} /><Route path="*" element={<Navigate to="/" replace />} /></Routes></BrowserRouter>
}
