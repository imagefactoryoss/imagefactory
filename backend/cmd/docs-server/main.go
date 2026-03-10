package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	stdhtml "html"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	gmhtml "github.com/yuin/goldmark/renderer/html"
)

const pageTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>%s</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;600&family=Source+Serif+4:opsz,wght@8..60,400;8..60,600&family=Space+Grotesk:wght@500;700&display=swap" rel="stylesheet">
  <style>
    :root {
      color-scheme: light;
      --bg: #f8fbff;
      --bg-2: #f1f5f9;
      --bg-3: #ecfeff;
      --ink: #0f172a;
      --muted: #475569;
      --border: rgba(148, 163, 184, 0.34);
      --panel: rgba(255, 255, 255, 0.92);
      --panel-2: #f8fafc;
      --accent: #0891b2;
      --accent-2: #2563eb;
      --accent-soft: rgba(8, 145, 178, 0.12);
      --accent-soft-2: rgba(37, 99, 235, 0.12);
      --code-bg: #0f172a;
      --code-ink: #e5e7eb;
      --shadow: 0 20px 50px rgba(15, 23, 42, 0.10);
      --toggle-bg: #dbeafe;
      --toggle-ink: #1e3a8a;
      --sidebar-ink: #0f172a;
    }
    [data-theme="dark"] {
      color-scheme: dark;
      --bg: #020617;
      --bg-2: #0f172a;
      --bg-3: #082f49;
      --ink: #e2e8f0;
      --muted: #94a3b8;
      --border: rgba(51, 65, 85, 0.92);
      --panel: rgba(15, 23, 42, 0.92);
      --panel-2: #172033;
      --accent: #22d3ee;
      --accent-2: #60a5fa;
      --accent-soft: rgba(34, 211, 238, 0.13);
      --accent-soft-2: rgba(96, 165, 250, 0.13);
      --code-bg: #09101c;
      --code-ink: #e5e7eb;
      --shadow: 0 24px 60px rgba(2, 6, 23, 0.45);
      --toggle-bg: #1e293b;
      --toggle-ink: #e5e7eb;
      --sidebar-ink: #f3f7fb;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      background:
        linear-gradient(135deg, var(--bg-2) 0%%, var(--bg) 52%%, var(--bg-3) 100%%);
      color: var(--ink);
      font-family: "Source Serif 4", Georgia, serif;
      line-height: 1.72;
      min-height: 100vh;
    }
    [data-theme="dark"] body {
      background:
        linear-gradient(135deg, var(--bg) 0%%, var(--bg-2) 56%%, #0b1120 100%%);
    }
    .layout {
      display: grid;
      grid-template-columns: 300px minmax(0, 1fr);
      gap: 28px;
      max-width: 1240px;
      margin: 0 auto;
      padding: 28px 20px 80px;
    }
    .sidebar {
      position: sticky;
      top: 18px;
      align-self: start;
      background: var(--panel);
      backdrop-filter: blur(16px);
      border: 1px solid var(--border);
      border-radius: 18px;
      padding: 18px 18px 24px;
      box-shadow: var(--shadow);
      max-height: calc(100vh - 36px);
      overflow-y: auto;
    }
    [data-theme="dark"] .sidebar {
      background: var(--panel);
    }
    .sidebar-brand {
      padding: 4px 2px 14px;
      margin-bottom: 6px;
    }
    .sidebar-mark {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      min-width: 38px;
      padding: 6px 10px;
      border-radius: 10px;
      background: var(--accent-2);
      color: #ffffff;
      font-family: "Space Grotesk", "IBM Plex Sans", "Segoe UI", system-ui, sans-serif;
      font-size: 0.78rem;
      font-weight: 700;
      letter-spacing: 0.12em;
      text-transform: uppercase;
      box-shadow: 0 10px 24px rgba(37, 99, 235, 0.22);
    }
    .sidebar-kicker {
      font-family: "Space Grotesk", "IBM Plex Sans", "Segoe UI", system-ui, sans-serif;
      font-size: 0.72rem;
      letter-spacing: 0.14em;
      text-transform: uppercase;
      color: var(--accent);
      margin: 12px 0 6px;
    }
    .sidebar-brand h1 {
      margin: 0;
      font-size: 1.22rem;
      line-height: 1.1;
      color: var(--sidebar-ink);
      font-family: "Space Grotesk", "IBM Plex Sans", "Segoe UI", system-ui, sans-serif;
    }
    .sidebar-brand p {
      margin: 8px 0 0;
      color: var(--muted);
      font-size: 0.92rem;
      line-height: 1.45;
      font-family: "IBM Plex Sans", "Segoe UI", system-ui, sans-serif;
    }
    .sidebar summary {
      font-size: 0.84rem;
      margin: 12px 0 8px;
      color: var(--muted);
      text-transform: uppercase;
      letter-spacing: 0.1em;
      cursor: pointer;
      list-style: none;
      font-family: "Space Grotesk", "IBM Plex Sans", "Segoe UI", system-ui, sans-serif;
    }
    .sidebar summary .section-title {
      display: inline-flex;
      align-items: center;
      gap: 8px;
      font-weight: 700;
    }
    .sidebar summary .section-title::before {
      content: "";
      width: 9px;
      height: 9px;
      border-radius: 999px;
      background: var(--accent-2);
      display: inline-block;
      box-shadow: 0 0 0 4px var(--accent-soft-2);
    }
    .sidebar summary .section-architecture::before { background: #2563eb; box-shadow: 0 0 0 4px rgba(37, 99, 235, 0.18); }
    .sidebar summary .section-build-management::before { background: #16a34a; box-shadow: 0 0 0 4px rgba(22, 163, 74, 0.18); }
    .sidebar summary .section-kubernetes-integration::before { background: #0ea5e9; box-shadow: 0 0 0 4px rgba(14, 165, 233, 0.18); }
    .sidebar summary .section-authentication-authorization::before { background: #9333ea; box-shadow: 0 0 0 4px rgba(147, 51, 234, 0.18); }
    .sidebar summary .section-user-journeys-role-based-guides::before { background: #6366f1; box-shadow: 0 0 0 4px rgba(99, 102, 241, 0.18); }
    .sidebar summary .section-testing-quality-assurance::before { background: #eab308; box-shadow: 0 0 0 4px rgba(234, 179, 8, 0.18); }
    .sidebar summary .section-reference-setup::before { background: #64748b; box-shadow: 0 0 0 4px rgba(100, 116, 139, 0.18); }
    .sidebar summary::-webkit-details-marker { display: none; }
    .sidebar details {
      border-top: 1px solid var(--border);
      padding-top: 8px;
      margin-top: 8px;
    }
    .sidebar ul {
      list-style: none;
      padding-left: 0;
      margin: 0 0 16px;
    }
    .sidebar li { margin: 6px 0; }
    .sidebar a {
      display: block;
      color: var(--sidebar-ink);
      text-decoration: none;
      font-size: 0.95rem;
      line-height: 1.35;
      padding: 8px 10px;
      border-radius: 10px;
      transition: background-color 120ms ease, color 120ms ease, transform 120ms ease;
      font-family: "IBM Plex Sans", "Segoe UI", system-ui, sans-serif;
    }
    .sidebar a:hover {
      color: var(--accent-2);
      background: linear-gradient(90deg, var(--accent-soft), transparent);
      text-decoration: none;
      transform: translateX(2px);
    }
    .sidebar a.current {
      background: linear-gradient(90deg, var(--accent-soft-2), transparent);
      color: var(--accent-2);
      font-weight: 700;
      box-shadow: inset 3px 0 0 var(--accent-2);
    }
    .content {
      background: var(--panel);
      backdrop-filter: blur(16px);
      border: 1px solid var(--border);
      border-radius: 18px;
      padding: 40px 48px;
      box-shadow: var(--shadow);
    }
    .prose {
      max-width: 76ch;
    }
    .topbar {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 16px;
      margin-bottom: 18px;
    }
    .title {
      font-size: 0.95rem;
      color: var(--sidebar-ink);
      letter-spacing: 0.08em;
      text-transform: uppercase;
      font-weight: 700;
      font-family: "Space Grotesk", "IBM Plex Sans", "Segoe UI", system-ui, sans-serif;
    }
    .theme-toggle {
      display: inline-flex;
      align-items: center;
      gap: 10px;
      border: 1px solid var(--border);
      background: var(--panel-2);
      color: var(--ink);
      padding: 6px 10px;
      border-radius: 999px;
      font-size: 0.85rem;
      cursor: pointer;
      font-family: "Space Grotesk", "IBM Plex Sans", "Segoe UI", system-ui, sans-serif;
    }
    .theme-toggle .pill {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      width: 30px;
      height: 18px;
      background: var(--toggle-bg);
      color: var(--toggle-ink);
      border-radius: 999px;
      font-size: 0.7rem;
      letter-spacing: 0.08em;
      text-transform: uppercase;
    }
    @media (max-width: 960px) {
      .layout { grid-template-columns: 1fr; }
      .sidebar { position: static; max-height: none; }
      .content { padding: 28px 24px; }
    }
    h1, h2, h3, h4, h5, h6 {
      margin-top: 1.6em;
      font-family: "Space Grotesk", "IBM Plex Sans", "Segoe UI", system-ui, sans-serif;
      line-height: 1.15;
      color: var(--sidebar-ink);
    }
    h1 {
      font-size: clamp(2.4rem, 4vw, 3.2rem);
      margin-top: 0;
      margin-bottom: 0.5rem;
      letter-spacing: -0.04em;
      text-wrap: balance;
    }
    h2 {
      font-size: 1.65rem;
      border-bottom: 1px solid var(--border);
      padding-bottom: 8px;
      letter-spacing: -0.025em;
    }
    h3 { font-size: 1.22rem; letter-spacing: -0.02em; }
    p { margin: 0.9em 0; font-size: 1.06rem; }
    li { font-size: 1.03rem; }
    a { color: var(--accent-2); text-decoration: none; }
    a:hover { text-decoration: underline; }
    blockquote {
      margin: 1.5em 0;
      padding: 14px 18px;
      border-left: 4px solid var(--accent-2);
      background: var(--panel-2);
      color: var(--ink);
      border-radius: 10px;
    }
    pre {
      background: var(--code-bg);
      color: var(--code-ink);
      padding: 16px;
      border-radius: 12px;
      overflow-x: auto;
      font-size: 0.9rem;
      box-shadow: inset 0 0 0 1px rgba(148, 163, 184, 0.08);
    }
    code {
      background: var(--panel-2);
      padding: 2px 6px;
      border-radius: 6px;
      font-family: "JetBrains Mono", "IBM Plex Mono", "SFMono-Regular", ui-monospace, monospace;
      font-size: 0.92em;
    }
    pre code { background: transparent; padding: 0; }
    table {
      border-collapse: collapse;
      width: 100%%;
      margin: 1.5em 0;
      border: 1px solid var(--border);
      border-radius: 12px;
      overflow: hidden;
    }
    th, td {
      border: 1px solid var(--border);
      padding: 10px 12px;
      text-align: left;
    }
    th {
      background: var(--panel-2);
      color: var(--ink);
      font-weight: 700;
    }
    tr:nth-child(even) { background: var(--panel-2); }
    ul, ol { padding-left: 1.5rem; }
    img { max-width: 100%%; height: auto; border-radius: 12px; }
    .prose img {
      border: 1px solid var(--border);
      box-shadow: var(--shadow);
    }
    .mermaid {
      margin: 1.8em 0;
      padding: 18px;
      border: 1px solid var(--border);
      border-radius: 16px;
      background: linear-gradient(180deg, var(--panel-2), transparent);
      box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.35);
      overflow-x: auto;
    }
    .footer {
      margin-top: 30px;
      font-size: 0.8rem;
      color: var(--muted);
      font-family: "IBM Plex Sans", "Segoe UI", system-ui, sans-serif;
    }
  </style>
  <script type="module">
    import mermaid from "https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs";
    mermaid.initialize({
      startOnLoad: false,
      theme: "base",
      themeVariables: {
        primaryColor: "#ffffff",
        primaryTextColor: "#0f172a",
        primaryBorderColor: "#cbd5f5",
        lineColor: "#94a3b8",
        secondaryColor: "#f1f5f9",
        tertiaryColor: "#e2e8f0",
        background: "#ffffff",
        fontFamily: "Space Grotesk, IBM Plex Sans, Segoe UI, system-ui, -apple-system, sans-serif"
      }
    });
    const blocks = document.querySelectorAll('pre code.language-mermaid, pre code.lang-mermaid');
    blocks.forEach((code) => {
      const pre = code.parentElement;
      const div = document.createElement('div');
      div.className = 'mermaid';
      div.textContent = code.textContent;
      pre.replaceWith(div);
    });
    mermaid.run({ querySelector: ".mermaid" });

    const detailsList = document.querySelectorAll('.sidebar details');
    const navLinks = document.querySelectorAll('.sidebar a');
    const KEY = 'docs.sidebar.open';
    const applyOpen = (title) => {
      let found = false;
      detailsList.forEach((details) => {
        const summary = details.querySelector('summary');
        const label = summary ? summary.textContent.trim() : '';
        if (title && label === title) {
          details.setAttribute('open', 'open');
          found = true;
        }
      });
      if (!found && detailsList.length > 0) {
        detailsList[0].setAttribute('open', 'open');
      }
    };

    const stored = localStorage.getItem(KEY);
    if (stored) {
      applyOpen(stored);
    } else {
      applyOpen(null);
    }

    detailsList.forEach((details) => {
      details.addEventListener('toggle', () => {
        if (details.open) {
          const summary = details.querySelector('summary');
          const label = summary ? summary.textContent.trim() : '';
          localStorage.setItem(KEY, label);
          detailsList.forEach((other) => {
            if (other !== details) {
              other.removeAttribute('open');
            }
          });
        }
      });
    });

    const currentPath = window.location.pathname;
    navLinks.forEach((link) => {
      const href = link.getAttribute('href');
      if (!href) return;
      if (href === currentPath || (currentPath === '/' && href === '/README.md')) {
        link.classList.add('current');
        const parentDetails = link.closest('details');
        if (parentDetails) {
          parentDetails.setAttribute('open', 'open');
          const summary = parentDetails.querySelector('summary');
          const label = summary ? summary.textContent.trim() : '';
          if (label) {
            localStorage.setItem(KEY, label);
          }
        }
      }
    });

    const THEME_KEY = 'docs.theme';
    const themeToggle = document.getElementById('theme-toggle');
    const themePill = document.getElementById('theme-pill');
    const applyTheme = (mode) => {
      if (mode === 'dark') {
        document.documentElement.setAttribute('data-theme', 'dark');
        themePill.textContent = 'Dark';
      } else {
        document.documentElement.removeAttribute('data-theme');
        themePill.textContent = 'Light';
      }
    };
    const storedTheme = localStorage.getItem(THEME_KEY);
    const preferredTheme = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
    applyTheme(storedTheme || preferredTheme);
    themeToggle.addEventListener('click', () => {
      const next = document.documentElement.getAttribute('data-theme') === 'dark' ? 'light' : 'dark';
      localStorage.setItem(THEME_KEY, next);
      applyTheme(next);
    });
  </script>
</head>
<body>
  <div class="layout">
    <aside class="sidebar">
      <div class="sidebar-brand">
        <div class="sidebar-mark">IF</div>
        <div class="sidebar-kicker">Image Factory</div>
        <h1>Documentation</h1>
        <p>Operator guides, product overviews, architecture reference, and contributor notes in one place.</p>
      </div>
      %s
    </aside>
    <main class="content">
      <div class="topbar">
        <div class="title">Image Factory Documentation</div>
        <button class="theme-toggle" id="theme-toggle" type="button">
          <span>Theme</span>
          <span class="pill" id="theme-pill">Light</span>
        </button>
      </div>
      <article class="prose">%s</article>
      <div class="footer">Image Factory documentation server · %s</div>
    </main>
  </div>
</body>
</html>`

const dirTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Index of %s</title>
  <style>
    body {
      margin: 0;
      padding: 32px 20px 60px;
      font-family: "Space Grotesk", "Segoe UI", system-ui, -apple-system, sans-serif;
      background: linear-gradient(135deg, #f1f5f9 0%%, #ffffff 56%%, #ecfeff 100%%);
      color: #0f172a;
    }
    .container {
      max-width: 920px;
      margin: 0 auto;
      background: rgba(255, 255, 255, 0.92);
      border: 1px solid rgba(148, 163, 184, 0.34);
      border-radius: 16px;
      padding: 28px 32px;
      box-shadow: 0 18px 40px rgba(15, 23, 42, 0.10);
      backdrop-filter: blur(16px);
    }
    h1 { margin-top: 0; }
    ul { list-style: none; padding: 0; }
    li { margin: 10px 0; }
    a { color: #2563eb; text-decoration: none; }
    a:hover { text-decoration: underline; }
    .meta { color: #475569; font-size: 0.85rem; margin-left: 8px; }
  </style>
</head>
<body>
  <div class="container">
    <h1>Index of %s</h1>
    <ul>%s</ul>
  </div>
</body>
</html>`

type docsServer struct {
	root          string
	indexFile     string
	markdown      goldmark.Markdown
	startedTime   time.Time
	autoUpdateNav bool
}

type navConfig struct {
	Sections []navSection `json:"sections"`
}

type navSection struct {
	Title string     `json:"title"`
	Items []navEntry `json:"items"`
}

type navEntry struct {
	Title  string `json:"title"`
	Path   string `json:"path"`
	Folder string `json:"folder"`
}

func main() {
	port := flag.Int("port", 8000, "Port to listen on")
	root := flag.String("root", "docs", "Root directory to serve")
	index := flag.String("index", "README.md", "Index markdown file name")
	autoUpdateNav := flag.Bool("auto-update-nav", true, "Auto-update _nav.json from docs on each request")
	flag.Parse()

	absRoot, err := filepath.Abs(*root)
	if err != nil {
		log.Fatalf("Failed to resolve docs root: %v", err)
	}
	if _, err := os.Stat(absRoot); err != nil {
		log.Fatalf("Docs root not found: %s", absRoot)
	}

	server := &docsServer{
		root:          absRoot,
		indexFile:     *index,
		autoUpdateNav: *autoUpdateNav,
		markdown: goldmark.New(
			goldmark.WithExtensions(extension.GFM),
			goldmark.WithParserOptions(parser.WithAutoHeadingID()),
			goldmark.WithRendererOptions(gmhtml.WithUnsafe()),
		),
		startedTime: time.Now(),
	}

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Docs server running at http://localhost%s", addr)
	log.Printf("Serving docs from: %s", absRoot)
	log.Fatal(http.ListenAndServe(addr, server))
}

func (s *docsServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cleanPath := path.Clean("/" + r.URL.Path)
	cleanPath = strings.TrimPrefix(cleanPath, "/")

	fullPath := filepath.Join(s.root, filepath.FromSlash(cleanPath))
	if !strings.HasPrefix(fullPath, s.root) {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if info.IsDir() {
		if s.tryServeIndex(w, r, fullPath, cleanPath) {
			return
		}
		s.renderDirectory(w, r, fullPath, cleanPath)
		return
	}

	if strings.EqualFold(filepath.Ext(fullPath), ".md") {
		s.renderMarkdown(w, r, fullPath, filepath.Base(fullPath))
		return
	}

	http.ServeFile(w, r, fullPath)
}

func (s *docsServer) tryServeIndex(w http.ResponseWriter, r *http.Request, dirPath, cleanPath string) bool {
	candidates := []string{s.indexFile, "README.md", "index.md"}
	for _, candidate := range candidates {
		filePath := filepath.Join(dirPath, candidate)
		if _, err := os.Stat(filePath); err == nil {
			title := candidate
			if cleanPath != "" {
				title = cleanPath
			}
			s.renderMarkdown(w, r, filePath, title)
			return true
		}
	}
	return false
}

func (s *docsServer) renderMarkdown(w http.ResponseWriter, r *http.Request, filePath, title string) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		http.Error(w, "Failed to read markdown", http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer
	if err := s.markdown.Convert(content, &buf); err != nil {
		http.Error(w, "Failed to render markdown", http.StatusInternalServerError)
		return
	}

	navHTML := s.buildNav()
	rendered := fmt.Sprintf(pageTemplate, stdhtml.EscapeString(title), navHTML, buf.String(), s.startedTime.Format(time.RFC1123))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(rendered))
}

func (s *docsServer) renderDirectory(w http.ResponseWriter, r *http.Request, dirPath, cleanPath string) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		http.Error(w, "Failed to list directory", http.StatusInternalServerError)
		return
	}

	type item struct {
		name  string
		isDir bool
		size  string
	}

	var items []item
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		items = append(items, item{
			name:  entry.Name(),
			isDir: entry.IsDir(),
			size:  fmt.Sprintf("%d bytes", info.Size()),
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].isDir != items[j].isDir {
			return items[i].isDir
		}
		return strings.ToLower(items[i].name) < strings.ToLower(items[j].name)
	})

	var b strings.Builder
	if cleanPath != "" {
		b.WriteString(`<li><a href="../">../</a></li>`)
	}
	for _, entry := range items {
		display := stdhtml.EscapeString(entry.name)
		link := path.Join("/", cleanPath, entry.name)
		if entry.isDir {
			display += "/"
			link += "/"
		}
		b.WriteString(fmt.Sprintf(`<li><a href="%s">%s</a><span class="meta">%s</span></li>`, link, display, entry.size))
	}

	title := "/" + cleanPath
	if cleanPath == "" {
		title = "/"
	}
	rendered := fmt.Sprintf(dirTemplate, stdhtml.EscapeString(title), stdhtml.EscapeString(title), b.String())
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(rendered))
}

func (s *docsServer) buildNav() string {
	if navHTML := s.buildNavFromJSON(); navHTML != "" {
		return navHTML
	}

	indexPath := filepath.Join(s.root, "DOCUMENTATION_INDEX.md")
	content, err := os.ReadFile(indexPath)
	if err != nil {
		return `<h2>Docs</h2><ul><li><a href="/README.md">README</a></li></ul>`
	}

	lines := strings.Split(string(content), "\n")
	currentSection := ""
	currentLocation := ""
	sections := make(map[string][]string)
	order := []string{
		"Build Management",
		"Kubernetes Integration",
		"Authentication & Authorization",
		"Routing & Navigation",
		"Admin Console",
		"Projects & Tenants",
		"Testing & Quality Assurance",
		"User Journeys & Role-Based Guides",
		"Planning",
		"Reference & Setup",
		"Status & Progress",
		"Fixes & Improvements",
		"Implementation",
		"Phases",
		"Reminders",
		"Documentation Meta",
	}

	linkRe := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "### ") {
			section := strings.TrimSpace(strings.TrimLeft(line, "#"))
			currentSection = section
			continue
		}

		if strings.HasPrefix(line, "**Location**:") {
			start := strings.Index(line, "`")
			end := strings.LastIndex(line, "`")
			if start >= 0 && end > start {
				currentLocation = strings.Trim(line[start+1:end], " /")
			}
			continue
		}

		if currentSection == "" {
			continue
		}

		if strings.HasPrefix(line, "|") && strings.Contains(line, "|") {
			cols := splitMarkdownRow(line)
			if len(cols) < 2 {
				continue
			}
			docCell := strings.TrimSpace(cols[0])
			if docCell == "" || strings.Contains(docCell, "---") || strings.HasPrefix(strings.ToLower(docCell), "various ") {
				continue
			}

			if match := linkRe.FindStringSubmatch(docCell); len(match) >= 3 {
				text := match[1]
				href := normalizeDocHref(match[2])
				sections[currentSection] = append(sections[currentSection], buildNavItem(text, href))
				continue
			}

			docName := strings.Trim(docCell, "*` ")
			href := docName
			if strings.Contains(docName, "/") {
				href = docName
			} else if currentLocation != "" {
				href = path.Join(currentLocation, docName)
			}
			href = normalizeDocHref(href)
			if href == "" {
				continue
			}
			sections[currentSection] = append(sections[currentSection], buildNavItem(docName, href))
			continue
		}

		matches := linkRe.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			if len(match) < 3 {
				continue
			}
			href := normalizeDocHref(match[2])
			if href == "" {
				continue
			}
			sections[currentSection] = append(sections[currentSection], buildNavItem(match[1], href))
		}
	}

	if len(sections) == 0 {
		return `<h2>Docs</h2><ul><li><a href="/README.md">README</a></li></ul>`
	}

	var b strings.Builder
	renderSection := func(title string, items []string) {
		if len(items) == 0 {
			return
		}
		b.WriteString("<details><summary>")
		b.WriteString(`<span class="section-title section-` + slugifySection(title) + `">`)
		b.WriteString(stdhtml.EscapeString(title))
		b.WriteString("</span></summary><ul>")
		for _, item := range items {
			b.WriteString(item)
		}
		b.WriteString("</ul></details>")
	}

	seen := make(map[string]bool)
	for _, title := range order {
		renderSection(title, sections[title])
		seen[title] = true
	}

	var remaining []string
	for title := range sections {
		if !seen[title] {
			remaining = append(remaining, title)
		}
	}
	sort.Strings(remaining)
	for _, title := range remaining {
		renderSection(title, sections[title])
	}

	return b.String()
}

func (s *docsServer) buildNavFromJSON() string {
	navPath := filepath.Join(s.root, "_nav.json")
	if s.autoUpdateNav {
		_ = s.updateNavJSON(navPath)
	}
	content, err := os.ReadFile(navPath)
	if err != nil {
		return ""
	}

	var cfg navConfig
	if err := json.Unmarshal(content, &cfg); err != nil {
		return ""
	}

	if len(cfg.Sections) == 0 {
		return ""
	}

	var b strings.Builder
	for _, section := range cfg.Sections {
		if section.Title == "" {
			continue
		}
		itemsHTML := s.renderNavItems(section.Items)
		if itemsHTML == "" {
			continue
		}
		b.WriteString("<details><summary>")
		b.WriteString(`<span class="section-title section-` + slugifySection(section.Title) + `">`)
		b.WriteString(stdhtml.EscapeString(section.Title))
		b.WriteString("</span></summary><ul>")
		b.WriteString(itemsHTML)
		b.WriteString("</ul></details>")
	}

	return b.String()
}

func (s *docsServer) updateNavJSON(navPath string) error {
	content, err := os.ReadFile(navPath)
	if err != nil {
		return err
	}

	var cfg navConfig
	if err := json.Unmarshal(content, &cfg); err != nil {
		return err
	}

	if len(cfg.Sections) == 0 {
		return nil
	}

	changed := false
	for i := range cfg.Sections {
		section := cfg.Sections[i]
		items := section.Items

		folderPrefixes := map[string]bool{}
		for _, item := range items {
			if item.Path == "" {
				continue
			}
			href := normalizeDocHref(item.Path)
			if href == "" {
				continue
			}
			folderPrefixes[path.Dir(href)] = true
		}

		filtered := make([]navEntry, 0, len(items))
		for _, item := range items {
			if item.Path == "" {
				filtered = append(filtered, item)
				continue
			}
			href := normalizeDocHref(item.Path)
			if href == "" {
				continue
			}
			target := filepath.Join(s.root, filepath.FromSlash(strings.TrimPrefix(href, "/")))
			if _, err := os.Stat(target); err == nil {
				item.Path = href
				filtered = append(filtered, item)
				continue
			}
			changed = true
		}

		existingPaths := map[string]bool{}
		for _, item := range filtered {
			if item.Path == "" {
				continue
			}
			existingPaths[normalizeDocHref(item.Path)] = true
		}

		folders := make([]string, 0, len(folderPrefixes))
		for folder := range folderPrefixes {
			folders = append(folders, folder)
		}
		sort.Strings(folders)

		for _, folder := range folders {
			folderPath := strings.TrimPrefix(folder, "/")
			folderFS := filepath.Join(s.root, filepath.FromSlash(folderPath))
			for _, md := range listMarkdownFiles(folderFS) {
				href := normalizeDocHref(path.Join(folder, md))
				if existingPaths[href] {
					continue
				}
				filtered = append(filtered, navEntry{
					Title: titleFromFilename(md),
					Path:  href,
				})
				existingPaths[href] = true
				changed = true
			}
		}

		section.Items = filtered
		cfg.Sections[i] = section
	}

	if !changed {
		return nil
	}

	updated, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	if bytes.Equal(bytes.TrimSpace(content), bytes.TrimSpace(updated)) {
		return nil
	}

	return os.WriteFile(navPath, append(updated, '\n'), 0644)
}

func listMarkdownFiles(folder string) []string {
	entries, err := os.ReadDir(folder)
	if err != nil {
		return nil
	}
	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(strings.ToLower(name), ".md") {
			files = append(files, name)
		}
	}
	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i]) < strings.ToLower(files[j])
	})
	return files
}

func titleFromFilename(filename string) string {
	base := strings.TrimSuffix(filename, filepath.Ext(filename))
	base = strings.ReplaceAll(base, "_", " ")
	base = strings.ReplaceAll(base, "-", " ")
	parts := strings.Fields(base)
	for i, p := range parts {
		runes := []rune(strings.ToLower(p))
		if len(runes) > 0 {
			runes[0] = unicode.ToUpper(runes[0])
		}
		parts[i] = string(runes)
	}
	return strings.Join(parts, " ")
}

func (s *docsServer) renderNavItems(items []navEntry) string {
	if len(items) == 0 {
		return ""
	}

	var b strings.Builder
	for _, item := range items {
		if item.Path != "" {
			href := normalizeDocHref(item.Path)
			if href == "" {
				continue
			}
			title := item.Title
			if title == "" {
				title = path.Base(href)
			}
			b.WriteString(buildNavItem(title, href))
			continue
		}

		if item.Folder != "" {
			folderItems := s.listFolderDocs(item.Folder)
			if len(folderItems) == 0 {
				continue
			}
			title := item.Title
			if title == "" {
				title = item.Folder
			}
			b.WriteString("<li>")
			b.WriteString(stdhtml.EscapeString(title))
			b.WriteString("<ul>")
			for _, entry := range folderItems {
				b.WriteString(buildNavItem(entry.Title, entry.Path))
			}
			b.WriteString("</ul></li>")
		}
	}

	return b.String()
}

type navItem struct {
	Title string
	Path  string
}

func (s *docsServer) listFolderDocs(folder string) []navItem {
	folderPath := filepath.Join(s.root, filepath.FromSlash(folder))
	entries, err := os.ReadDir(folderPath)
	if err != nil {
		return nil
	}
	var items []navItem
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".md") {
			continue
		}
		href := path.Join("/", folder, name)
		items = append(items, navItem{
			Title: strings.TrimSuffix(name, filepath.Ext(name)),
			Path:  href,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].Title) < strings.ToLower(items[j].Title)
	})
	return items
}

func slugifySection(title string) string {
	lower := strings.ToLower(title)
	var b strings.Builder
	lastDash := false
	for _, r := range lower {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteRune('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "section"
	}
	return out
}

func normalizeDocHref(href string) string {
	if strings.HasPrefix(href, "http") || strings.HasPrefix(href, "#") {
		return ""
	}
	href = strings.TrimPrefix(href, "docs/")
	href = strings.TrimPrefix(href, "./")
	href = path.Clean("/" + href)
	return href
}

func buildNavItem(text, href string) string {
	var b strings.Builder
	b.WriteString(`<li><a href="`)
	b.WriteString(stdhtml.EscapeString(href))
	b.WriteString(`">`)
	b.WriteString(stdhtml.EscapeString(text))
	b.WriteString("</a></li>")
	return b.String()
}

func splitMarkdownRow(line string) []string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || !strings.HasPrefix(trimmed, "|") {
		return nil
	}
	trimmed = strings.TrimPrefix(trimmed, "|")
	trimmed = strings.TrimSuffix(trimmed, "|")
	parts := strings.Split(trimmed, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}
