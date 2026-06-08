package main

import (
	_ "embed"
	"flag"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/Fiecher/searchinator"
	"github.com/Fiecher/searchinator/internal/docload"
	"github.com/Fiecher/searchinator/internal/sampledata"
	"github.com/Fiecher/searchinator/pkg/analysis"
	"github.com/Fiecher/searchinator/pkg/engine"
	"github.com/Fiecher/searchinator/pkg/highlight"
	"github.com/Fiecher/searchinator/pkg/index"
	"github.com/Fiecher/searchinator/pkg/ranking"
	"github.com/Fiecher/searchinator/pkg/semantic"
)

//go:embed icon.svg
var iconSVG []byte

var appIcon = fyne.NewStaticResource("searchinator.svg", iconSVG)

type ui struct {
	corpus []searchinator.Document

	bm25  *engine.Engine
	tfidf *engine.Engine
	fuzzy *engine.Engine

	predictor semantic.WordPredictor

	analyzer analysis.Analyzer

	ranker     string
	useFuzzy   bool
	boolMode   bool
	fuzzyDist  int
	dataDir    string
	bm25Params ranking.BM25Params

	results     []searchinator.Result
	resultSpans [][]highlight.Span
	occurrences int
	query       string
	predicted   []string
	busy        bool

	resultsBox *fyne.Container
	importBtn  *widget.Button
	progress   *widget.ProgressBarInfinite
	status     *widget.Label
}

func main() {
	data := flag.String("data", "", "directory for a durable segmented index; empty = in-memory only")
	flag.Parse()

	u := &ui{ranker: "BM25", fuzzyDist: 1, dataDir: *data, bm25Params: ranking.DefaultBM25Params()}

	var err error
	bm25cfg := searchConfig(ranking.NewBM25(u.bm25Params))
	if *data != "" {

		idx, oerr := index.OpenSegmented(*data)
		if oerr != nil {
			fatal(oerr)
		}
		if u.bm25, err = engine.NewEngineWithIndex(bm25cfg, idx); err != nil {
			fatal(err)
		}
		u.corpus = docsFromIndex(idx)
		if len(u.corpus) == 0 {
			u.corpus = sampledata.Corpus()
			if err = u.bm25.Index(u.corpus); err != nil {
				fatal(err)
			}
			if err = u.bm25.Flush(); err != nil {
				fatal(err)
			}
		}
	} else {
		u.corpus = sampledata.Corpus()
		if u.bm25, err = engine.NewEngine(bm25cfg); err != nil {
			fatal(err)
		}
		if err = u.bm25.Index(u.corpus); err != nil {
			fatal(err)
		}
	}

	if u.tfidf, err = engine.NewEngine(searchConfig(ranking.NewTFIDF())); err != nil {
		fatal(err)
	}
	if err = u.tfidf.Index(u.corpus); err != nil {
		fatal(err)
	}

	u.analyzer = searchAnalyzer()
	u.predictor = semantic.NewCorpusPredictor(u.corpus, u.analyzer.Analyze)

	a := app.New()
	a.SetIcon(appIcon)
	w := a.NewWindow("searchinator demo")
	w.SetIcon(appIcon)
	w.Resize(fyne.NewSize(760, 560))
	w.SetContent(u.build(w))
	w.ShowAndRun()
}

func (u *ui) build(w fyne.Window) fyne.CanvasObject {
	entry := widget.NewEntry()

	entry.SetPlaceHolder("Type a query, e.g. " + sampledata.ExampleQueryFrom(u.corpus))

	search := func() { u.runSearch(entry.Text) }
	entry.OnSubmitted = func(string) { search() }

	searchBtn := widget.NewButtonWithIcon("Search", theme.SearchIcon(), search)
	searchBtn.Importance = widget.HighImportance

	var fuzzyChk, boolChk *widget.Check
	rankerSel := widget.NewSelect([]string{"BM25", "TF-IDF", "Semantic"}, func(v string) {
		u.ranker = v

		if v == "Semantic" {
			u.useFuzzy, u.boolMode = false, false
			fuzzyChk.SetChecked(false)
			boolChk.SetChecked(false)
			fuzzyChk.Disable()
			boolChk.Disable()
		} else {
			fuzzyChk.Enable()
			boolChk.Enable()
		}
		if u.query != "" {
			u.runSearch(u.query)
		}
	})

	fuzzyChk = widget.NewCheck("Fuzzy", func(on bool) {
		u.useFuzzy = on
		if u.query != "" {
			u.runSearch(u.query)
		}
	})
	boolChk = widget.NewCheck("Boolean mode", func(on bool) {
		u.boolMode = on

		if on {
			entry.SetPlaceHolder(`e.g. (go OR rust) AND year>=2010 NOT "garbage collector"`)
		} else {
			entry.SetPlaceHolder("Type a query, e.g. " + sampledata.ExampleQueryFrom(u.corpus))
		}
		if u.query != "" {
			u.runSearch(u.query)
		}
	})

	rankerSel.SetSelected("BM25")

	u.resultsBox = container.NewVBox()

	u.importBtn = widget.NewButtonWithIcon("Import…", theme.FolderOpenIcon(), func() { u.importFile(w) })
	docsBtn := widget.NewButtonWithIcon("Documents…", theme.StorageIcon(), func() { u.showDocuments(w) })
	settingsBtn := widget.NewButtonWithIcon("Settings…", theme.SettingsIcon(), func() { u.showSettings(w) })
	helpBtn := widget.NewButtonWithIcon("Help", theme.HelpIcon(), func() { u.showHelp(w) })

	u.progress = widget.NewProgressBarInfinite()
	u.progress.Hide()

	u.status = widget.NewLabel("")
	u.status.Wrapping = fyne.TextWrapWord
	u.setStatus()
	u.renderResults()

	queryBar := container.NewBorder(nil, nil, nil, searchBtn, entry)

	modeControls := container.NewHBox(
		widget.NewLabel("Ranker:"), rankerSel,
		widget.NewSeparator(),
		fuzzyChk, boolChk,
	)
	actions := container.NewHBox(u.importBtn, docsBtn, settingsBtn, helpBtn)
	controls := container.NewBorder(nil, nil, nil, actions, modeControls)

	top := container.NewVBox(queryBar, controls, widget.NewSeparator())
	bottom := container.NewVBox(u.progress, u.status)

	return container.NewBorder(top, bottom, nil, nil, container.NewVScroll(u.resultsBox))
}

const helpMarkdown = "" +
	"# Search syntax\n\n" +
	"## Plain search\n" +
	"Type one or more words; results are ranked by relevance (BM25 or TF-IDF).\n\n" +
	"- `memory safety` — documents mentioning either word, best matches first\n\n" +
	"## Semantic\n" +
	"Pick **Semantic** to search by *description* instead of exact words. " +
	"Your text is sent to a word predictor, which proposes the words most " +
	"likely to appear in matching documents; those predicted words are then " +
	"searched and highlighted, and listed in the status bar. Fuzzy and " +
	"Boolean mode do not apply here.\n\n" +
	"- `язык со сборкой мусора` — predicts *сборщика, владения, …* and finds them\n" +
	"- The default predictor is offline (corpus co-occurrence plus a small " +
	"thesaurus); a real LLM can be plugged in without changing the workflow.\n\n" +
	"## Fuzzy\n" +
	"Tick **Fuzzy** to tolerate typos: `memroy` still finds *memory*.\n\n" +
	"## Boolean mode\n" +
	"Tick **Boolean mode** to enable operators, phrases and filters. " +
	"Operators must be UPPERCASE.\n\n" +
	"- `AND` — both terms (a plain space means AND too): `go AND fast`\n" +
	"- `OR` — either term: `rust OR swift`\n" +
	"- `NOT` — exclude a term: `language NOT java`\n" +
	"- `( )` — group sub-expressions: `(go OR rust) AND fast`\n" +
	"- `\"…\"` — exact phrase, words must be adjacent: `\"garbage collector\"`\n\n" +
	"## Metadata filters (Boolean mode)\n" +
	"Write `field OP value`. OP is one of  `=`  `!=`  `>`  `<`  `>=`  `<=`.\n" +
	"Numeric compares need numbers on both sides; `=` and `!=` also work on text.\n\n" +
	"- `year>=2010` — published in 2010 or later\n" +
	"- `year<2015` — before 2015\n" +
	"- `paradigm=functional` — exact field match\n" +
	"- `paradigm!=object-oriented` — exclude a field value\n\n" +
	"## Combine everything\n" +
	"`(go OR rust) AND year>=2010 NOT \"garbage collector\"`\n\n" +
	"## Settings\n" +
	"Open **Settings…** to choose where the index is stored and to tune ranking:\n\n" +
	"- **Index storage** — pick a folder for a durable on-disk index, or leave " +
	"empty for an in-memory index. A non-empty folder with an existing index is " +
	"loaded as the working corpus; otherwise the current corpus is written to it.\n" +
	"- **Fuzzy distance** — maximum edit distance for fuzzy matching (1 or 2).\n" +
	"- **BM25 k1 / b** — term-frequency saturation and length normalisation.\n\n" +
	"Applying settings rebuilds the engines without restarting the program.\n"

func (u *ui) showHelp(w fyne.Window) {
	rt := widget.NewRichTextFromMarkdown(helpMarkdown)
	rt.Wrapping = fyne.TextWrapWord
	scroll := container.NewVScroll(rt)
	scroll.SetMinSize(fyne.NewSize(540, 480))
	dialog.ShowCustom("Query help", "Got it", scroll, w)
}

func (u *ui) storageSummary() string {
	if u.dataDir == "" {
		return "Using an in-memory index (changes are not persisted)."
	}
	return "Durable segmented index at " + u.dataDir + " (changes are flushed to disk)."
}

func (u *ui) showSettings(w fyne.Window) {
	storageEntry := widget.NewEntry()
	storageEntry.SetPlaceHolder("(in-memory — durable index disabled)")
	storageEntry.SetText(u.dataDir)

	browseBtn := widget.NewButtonWithIcon("Browse…", theme.FolderOpenIcon(), func() {
		if nativeFileDialogAvailable {
			path, ok, err := nativeOpenFolder("Choose a folder for the durable index")
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if ok {
				storageEntry.SetText(path)
			}
			return
		}
		dialog.ShowFolderOpen(func(lu fyne.ListableURI, err error) {
			if err != nil || lu == nil {
				return
			}
			storageEntry.SetText(lu.Path())
		}, w)
	})
	memBtn := widget.NewButton("In-memory", func() { storageEntry.SetText("") })
	storageRow := container.NewBorder(nil, nil, nil,
		container.NewHBox(browseBtn, memBtn), storageEntry)

	fuzzySel := widget.NewSelect([]string{"1", "2"}, nil)
	fuzzySel.SetSelected(strconv.Itoa(u.fuzzyDist))

	k1Entry := widget.NewEntry()
	k1Entry.SetText(strconv.FormatFloat(u.bm25Params.K1, 'g', -1, 64))
	bEntry := widget.NewEntry()
	bEntry.SetText(strconv.FormatFloat(u.bm25Params.B, 'g', -1, 64))

	form := widget.NewForm(
		widget.NewFormItem("Index storage", storageRow),
		widget.NewFormItem("Fuzzy distance", fuzzySel),
		widget.NewFormItem("BM25 k1", k1Entry),
		widget.NewFormItem("BM25 b", bEntry),
	)
	form.Items[0].HintText = "Folder for a durable segmented index; empty = in-memory."
	form.Items[2].HintText = "Term-frequency saturation (typical 1.2–2.0)."
	form.Items[3].HintText = "Length normalisation, 0–1 (typical 0.75)."

	content := container.NewVScroll(form)
	content.SetMinSize(fyne.NewSize(640, 380))

	dialog.ShowCustomConfirm("Settings", "Apply", "Cancel", content, func(ok bool) {
		if !ok {
			return
		}
		k1, err := strconv.ParseFloat(strings.TrimSpace(k1Entry.Text), 64)
		if err != nil {
			dialog.ShowError(fmt.Errorf("invalid BM25 k1: %v", err), w)
			return
		}
		b, err := strconv.ParseFloat(strings.TrimSpace(bEntry.Text), 64)
		if err != nil {
			dialog.ShowError(fmt.Errorf("invalid BM25 b: %v", err), w)
			return
		}
		if k1 < 0 || b < 0 || b > 1 {
			dialog.ShowError(fmt.Errorf("k1 must be ≥ 0 and b in [0, 1]"), w)
			return
		}
		dist := 1
		if fuzzySel.Selected != "" {
			dist, _ = strconv.Atoi(fuzzySel.Selected)
		}
		dir := strings.TrimSpace(storageEntry.Text)

		u.fuzzyDist = dist
		u.setBusy(true)
		u.status.SetText("Applying settings …")
		go func() {
			err := u.rebuildAll(dir, ranking.BM25Params{K1: k1, B: b})
			fyne.Do(func() {
				u.setBusy(false)
				if err != nil {
					dialog.ShowError(err, w)
					u.setStatus()
					return
				}
				u.setStatus()
				if u.query != "" {
					u.runSearch(u.query)
				}
				dialog.ShowInformation("Settings applied", u.storageSummary(), w)
			})
		}()
	}, w)
}

func (u *ui) rebuildAll(dir string, params ranking.BM25Params) error {
	cfg := searchConfig(ranking.NewBM25(params))
	corpus := u.corpus

	var bm *engine.Engine
	var err error
	if dir != "" {
		idx, oerr := index.OpenSegmented(dir)
		if oerr != nil {
			return oerr
		}
		if bm, err = engine.NewEngineWithIndex(cfg, idx); err != nil {
			return err
		}
		if existing := docsFromIndex(idx); len(existing) > 0 {
			corpus = existing
		} else {
			if err = bm.Index(corpus); err != nil {
				return err
			}
			if err = bm.Flush(); err != nil {
				return err
			}
		}
	} else {
		if bm, err = engine.NewEngine(cfg); err != nil {
			return err
		}
		if err = bm.Index(corpus); err != nil {
			return err
		}
	}

	tf, err := engine.NewEngine(searchConfig(ranking.NewTFIDF()))
	if err != nil {
		return err
	}
	if err = tf.Index(corpus); err != nil {
		return err
	}

	u.corpus = corpus
	u.bm25, u.tfidf = bm, tf
	u.predictor = semantic.NewCorpusPredictor(corpus, u.analyzer.Analyze)
	u.bm25Params = params
	u.dataDir = dir
	u.fuzzy = nil
	return nil
}

func (u *ui) setBusy(b bool) {
	u.busy = b
	if b {
		u.progress.Show()
		u.importBtn.Disable()
	} else {
		u.progress.Hide()
		u.importBtn.Enable()
	}
}

func (u *ui) runSearch(query string) {
	if u.busy {
		return
	}
	u.query = query
	if query == "" {
		u.results, u.resultSpans, u.occurrences = nil, nil, 0
		u.renderResults()
		u.setStatus()
		return
	}

	boolMode := u.boolMode
	semanticMode := u.semanticMode()
	u.setBusy(true)
	u.status.SetText(fmt.Sprintf("Searching the index for %q (%s) …", query, u.modeLabel()))

	go func() {
		var (
			results   []searchinator.Result
			err       error
			predicted []string
		)

		eng := u.engineFor()
		hl := eng
		hlQuery := query
		switch {
		case semanticMode:
			hl = u.bm25
			predicted, err = u.predictor.Predict(query, 6)
			if err == nil {
				hlQuery = strings.Join(predicted, " ")
				if strings.TrimSpace(hlQuery) == "" {
					hlQuery = query
				}
				results, err = u.bm25.Search(hlQuery)
			}
		case boolMode:
			results, err = eng.SearchBool(query)
		default:
			results, err = eng.Search(query)
		}

		var (
			spans = make([][]highlight.Span, 0, len(results))
			occ   int
		)
		if err == nil {
			for _, r := range results {
				occ += hl.TermOccurrences(r.Document.ID, hlQuery)
				sp, _ := hl.Highlights(r.Document.ID, hlQuery)
				spans = append(spans, sp)
			}
		}

		fyne.Do(func() {
			u.setBusy(false)
			if err != nil {
				u.results, u.resultSpans, u.occurrences, u.predicted = nil, nil, 0, nil
				u.renderResults()
				u.status.SetText("Error: " + err.Error())
				return
			}
			u.results = results
			u.resultSpans = spans
			u.occurrences = occ
			u.predicted = predicted
			u.status.SetText(fmt.Sprintf("Found %d matches in %d documents — rendering …", occ, len(results)))
			u.renderResults()
			u.setStatus()
		})
	}()
}

func (u *ui) semanticMode() bool { return u.ranker == "Semantic" }

func (u *ui) modeLabel() string {
	if u.semanticMode() {
		return "Semantic (predicted words)"
	}
	mode := u.ranker
	if u.useFuzzy {
		mode = "BM25 + fuzzy"
	}
	if u.boolMode {
		mode += ", boolean"
	}
	return mode
}

func (u *ui) renderResults() {
	u.resultsBox.RemoveAll()
	if len(u.results) == 0 {
		hint := "No results."
		if u.query == "" {
			hint = "Type a query and press Enter."
		}
		u.resultsBox.Add(widget.NewLabel(hint))
		u.resultsBox.Refresh()
		return
	}
	for i, r := range u.results {
		title := widget.NewLabelWithStyle(
			fmt.Sprintf("#%d  %s   score %.4f   %s", i+1, r.Document.ID, r.Score, metaTag(r.Document.Meta)),
			fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		var spans []highlight.Span
		if i < len(u.resultSpans) {
			spans = u.resultSpans[i]
		} else {
			spans = u.spansFor(r.Document.ID)
		}
		snippet := widget.NewRichText(spansToSegments(spans)...)
		snippet.Wrapping = fyne.TextWrapWord
		u.resultsBox.Add(container.NewVBox(title, snippet, widget.NewSeparator()))
	}
	u.resultsBox.Refresh()
}

func spansToSegments(spans []highlight.Span) []widget.RichTextSegment {
	segs := make([]widget.RichTextSegment, 0, len(spans))
	for _, sp := range spans {
		seg := &widget.TextSegment{Text: sp.Text, Style: widget.RichTextStyle{Inline: true}}
		if sp.Match {
			seg.Style.TextStyle = fyne.TextStyle{Bold: true}
			seg.Style.ColorName = theme.ColorNamePrimary
		}
		segs = append(segs, seg)
	}
	if len(segs) == 0 {
		segs = append(segs, &widget.TextSegment{Style: widget.RichTextStyle{Inline: true}})
	}
	return segs
}

func (u *ui) importFile(w fyne.Window) {
	if nativeFileDialogAvailable {
		path, ok, err := nativeOpenFile("Import a document to index", "Documents", docload.SupportedExtensions)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		if !ok {
			return
		}
		u.startImport(path, w)
		return
	}

	fd := dialog.NewFileOpen(func(rc fyne.URIReadCloser, err error) {
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		if rc == nil {
			return
		}
		path := rc.URI().Path()
		rc.Close()
		u.startImport(path, w)
	}, w)
	fd.SetFilter(storage.NewExtensionFileFilter(docload.SupportedExtensions))
	fd.Show()
}

func (u *ui) startImport(path string, w fyne.Window) {
	if u.busy {
		return
	}
	u.setBusy(true)
	u.status.SetText("Indexing " + filepath.Base(path) + " …")

	go func() {
		doc, err := docload.Load(path)
		if err == nil && doc.Text == "" {
			err = fmt.Errorf("no text extracted from %q", filepath.Base(path))
		}
		if err == nil {
			err = u.addDocument(doc)
		}

		fyne.Do(func() {
			u.setBusy(false)
			if err != nil {
				dialog.ShowError(err, w)
				u.setStatus()
				return
			}
			u.setStatus()
			if u.query != "" {
				u.runSearch(u.query)
			}
			dialog.ShowInformation("Imported",
				fmt.Sprintf("Indexed %q (%d chars).", doc.ID, len([]rune(doc.Text))), w)
		})
	}()
}

func (u *ui) addDocument(doc searchinator.Document) error {
	if err := u.bm25.Index([]searchinator.Document{doc}); err != nil {
		return err
	}
	if err := u.tfidf.Index([]searchinator.Document{doc}); err != nil {
		return err
	}
	u.fuzzy = nil

	replaced := false
	for i := range u.corpus {
		if u.corpus[i].ID == doc.ID {
			u.corpus[i] = doc
			replaced = true
			break
		}
	}
	if !replaced {
		u.corpus = append(u.corpus, doc)
	}

	if cp, ok := u.predictor.(*semantic.CorpusPredictor); ok && !replaced {
		cp.Index([]searchinator.Document{doc})
	} else {
		u.predictor = semantic.NewCorpusPredictor(u.corpus, u.analyzer.Analyze)
	}

	if u.dataDir != "" {
		return u.bm25.Flush()
	}
	return nil
}

func (u *ui) showDocuments(w fyne.Window) {
	list := container.NewVBox()

	var refresh func()
	refresh = func() {
		list.RemoveAll()
		if len(u.corpus) == 0 {
			list.Add(widget.NewLabel("No documents."))
			list.Refresh()
			return
		}
		docs := append([]searchinator.Document{}, u.corpus...)
		sort.Slice(docs, func(i, j int) bool { return docs[i].ID < docs[j].ID })
		for _, d := range docs {
			id := d.ID
			label := widget.NewLabel(fmt.Sprintf("%s   —   %s", id, docTag(d)))
			del := widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {
				if u.busy {
					dialog.ShowInformation("Busy", "An import is still indexing; try again in a moment.", w)
					return
				}
				if err := u.removeDocument(id); err != nil {
					dialog.ShowError(err, w)
					return
				}
				u.setStatus()
				if u.query != "" {
					u.runSearch(u.query)
				}
				refresh()
			})
			del.Importance = widget.DangerImportance
			list.Add(container.NewBorder(nil, nil, nil, del, label))
		}
		list.Refresh()
	}
	refresh()

	scroll := container.NewVScroll(list)
	scroll.SetMinSize(fyne.NewSize(460, 380))
	dialog.ShowCustom("Loaded documents", "Close", scroll, w)
}

func (u *ui) removeDocument(id string) error {
	if err := u.bm25.Delete(id); err != nil {
		return err
	}
	_ = u.tfidf.Delete(id)
	u.fuzzy = nil

	for i := range u.corpus {
		if u.corpus[i].ID == id {
			u.corpus = append(u.corpus[:i], u.corpus[i+1:]...)
			break
		}
	}

	u.predictor = semantic.NewCorpusPredictor(u.corpus, u.analyzer.Analyze)
	return nil
}

func docTag(d searchinator.Document) string {
	size := fmt.Sprintf("%d chars", len([]rune(d.Text)))
	if d.Meta != nil {
		if format, ok := d.Meta["format"].(string); ok && format != "" {
			return fmt.Sprintf("imported %s, %s", format, size)
		}
		if year, ok := d.Meta["year"].(int); ok {
			return fmt.Sprintf("sample, year %d, %s", year, size)
		}
	}
	return size
}

func docsFromIndex(idx *index.SegmentedIndex) []searchinator.Document {
	ids := idx.DocumentIDs()
	docs := make([]searchinator.Document, 0, len(ids))
	for _, id := range ids {
		if d, ok := idx.GetDocument(id); ok {
			docs = append(docs, d)
		}
	}
	return docs
}

func (u *ui) engineFor() *engine.Engine {
	if u.useFuzzy {
		if u.fuzzy == nil {
			u.fuzzy = u.buildFuzzy()
		}
		return u.fuzzy
	}
	if u.ranker == "TF-IDF" {
		return u.tfidf
	}
	return u.bm25
}

func (u *ui) buildFuzzy() *engine.Engine {
	vocab := engine.VocabularyFromIndex(u.bm25)
	cfg := engine.Config{
		Analyzer: analysis.NewPipelineAnalyzer(
			analysis.NewWhitespaceTokenizer(),
			analysis.NewLowercaseFilter(),
			analysis.NewPunctuationFilter(),
			analysis.NewStopWordsFilter(analysis.DefaultEnglishStopWords()),
			analysis.NewStopWordsFilter(analysis.DefaultRussianStopWords()),
			analysis.NewPorterStemmer(),
			analysis.NewRussianStemmer(),
			analysis.NewFuzzyFilter(vocab, u.fuzzyDist),
		),
		Ranker: ranking.NewBM25(ranking.DefaultBM25Params()),
	}
	fe, err := engine.NewEngine(cfg)
	if err != nil {
		fatal(err)
	}
	if err := fe.Index(u.corpus); err != nil {
		fatal(err)
	}
	return fe
}

func (u *ui) spansFor(docID string) []highlight.Span {
	if sp, ok := u.engineFor().Highlights(docID, u.query); ok && len(sp) > 0 {
		return sp
	}
	if doc, ok := findDoc(u.corpus, docID); ok {
		return []highlight.Span{{Text: truncate(doc.Text, 160)}}
	}
	return nil
}

func (u *ui) setStatus() {
	s := u.bm25.Stats()
	base := fmt.Sprintf("Mode: %s   |   %d results, %d words found   |   corpus: %d docs, %d terms, avg %.1f tokens",
		u.modeLabel(), len(u.results), u.occurrences, s.DocumentCount, s.TermCount, s.AverageDocumentLength)
	if u.semanticMode() && u.query != "" {
		if len(u.predicted) > 0 {
			base += "   |   predicted words: " + strings.Join(u.predicted, ", ")
		} else {
			base += "   |   predicted words: (none)"
		}
	}
	u.status.SetText(base)
}

func searchAnalyzer() analysis.Analyzer {
	return analysis.NewPipelineAnalyzer(
		analysis.NewWhitespaceTokenizer(),
		analysis.NewLowercaseFilter(),
		analysis.NewPunctuationFilter(),
		analysis.NewStopWordsFilter(analysis.DefaultEnglishStopWords()),
		analysis.NewStopWordsFilter(analysis.DefaultRussianStopWords()),
		analysis.NewPorterStemmer(),
		analysis.NewRussianStemmer(),
	)
}

func searchConfig(ranker ranking.Ranker) engine.Config {
	return engine.Config{Analyzer: searchAnalyzer(), Ranker: ranker}
}

func metaTag(meta map[string]any) string {
	if meta == nil {
		return ""
	}
	if year, ok := meta["year"].(int); ok {
		return fmt.Sprintf("(%d)", year)
	}
	if format, ok := meta["format"].(string); ok && format != "" {
		return "(" + format + ")"
	}
	return ""
}

func findDoc(docs []searchinator.Document, id string) (searchinator.Document, bool) {
	for _, d := range docs {
		if d.ID == id {
			return d, true
		}
	}
	return searchinator.Document{}, false
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "..."
}

func fatal(err error) {
	panic(err)
}
