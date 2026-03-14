package images

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	_ "golang.org/x/image/webp"

	tea "charm.land/bubbletea/v2"
	"github.com/blacktop/go-termimg"
	"github.com/dotBeeps/noms/internal/ui/theme"
)

var debugLog *log.Logger

var httpClient = &http.Client{Timeout: 10 * time.Second}

func init() {
	f, err := os.OpenFile("/tmp/noms-images-debug.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		debugLog = log.New(os.Stderr, "[img] ", log.Ltime|log.Lmicroseconds)
	} else {
		debugLog = log.New(f, "[img] ", log.Ltime|log.Lmicroseconds)
	}
}

// ImageFetchedMsg signals that an image has been downloaded and cached.
type ImageFetchedMsg struct {
	URL string
}

// placementInfo tracks the last virtual placement dimensions for an image.
type placementInfo struct {
	cols     int
	rows     int
	imageNum uint32
}

// imageNumCounter is a global counter for assigning unique Kitty image IDs.
var imageNumCounter uint32

// Cache manages downloaded images for terminal graphics rendering.
// For Kitty, transmits image data via /dev/tty and returns cellbuf-compatible
// Unicode placeholders. For other protocols, uses go-termimg widgets directly.
type Cache struct {
	files      sync.Map // url -> filepath
	widgets    sync.Map // url -> *termimg.ImageWidget (non-Kitty protocols)
	pending    sync.Map // url -> bool (in-flight downloads)
	imageNums  sync.Map // url -> uint32 (Kitty: transmitted image IDs)
	placements sync.Map // url -> placementInfo (Kitty: last placement dimensions)
	dir        string
	protocol   termimg.Protocol
	tty        *os.File // direct terminal access for Kitty APC bypass

	mu      sync.Mutex
	lastErr error
}

// New creates a new image cache. Detects the best available graphics protocol.
func New() *Cache {
	protocol := termimg.DetectProtocol()
	dir := filepath.Join(os.TempDir(), "noms-images")
	os.MkdirAll(dir, 0o700) //nolint:errcheck

	var tty *os.File
	if protocol == termimg.Kitty {
		var err error
		tty, err = os.OpenFile("/dev/tty", os.O_WRONLY, 0)
		if err != nil {
			debugLog.Printf("New: failed to open /dev/tty, falling back to halfblocks: %v", err)
			protocol = termimg.Halfblocks
		}
	}

	debugLog.Printf("New: protocol=%s(%d) enabled=%v dir=%s kitty=%v", protocol, protocol, protocol != termimg.Unsupported, dir, tty != nil)
	return &Cache{dir: dir, protocol: protocol, tty: tty}
}

// Enabled returns whether any graphics protocol is supported.
func (c *Cache) Enabled() bool {
	return c != nil && c.protocol != termimg.Unsupported
}

// Protocol returns the detected graphics protocol name.
func (c *Cache) Protocol() string {
	if c == nil {
		return "none"
	}
	return c.protocol.String()
}

// IsCached returns whether the given URL has been downloaded.
func (c *Cache) IsCached(url string) bool {
	if c == nil {
		return false
	}
	_, ok := c.files.Load(url)
	return ok
}

// Fetch returns a tea.Cmd that downloads an image and caches it.
// Returns nil if cache is disabled, URL is empty, already cached, or already fetching.
func Fetch(c *Cache, url string) tea.Cmd {
	return fetchWithTransform(c, url, nil)
}

// FetchAvatar returns a tea.Cmd that downloads an avatar with rounded corners.
func FetchAvatar(c *Cache, url string) tea.Cmd {
	return fetchWithTransform(c, url, func(img image.Image) image.Image {
		return applyRoundedCorners(img, 0.3, theme.ANSIToRGB(theme.SurfaceCode()))
	})
}

func (c *Cache) FetchAvatar(url string) tea.Cmd {
	return fetchWithTransform(c, url, func(img image.Image) image.Image {
		return applyRoundedCorners(img, 0.3, theme.ANSIToRGB(theme.SurfaceCode()))
	})
}

func fetchWithTransform(c *Cache, url string, transform func(image.Image) image.Image) tea.Cmd {
	if c == nil || !c.Enabled() || url == "" {
		debugLog.Printf("Fetch: skip url=%q (nil=%v enabled=%v empty=%v)", truncateURL(url), c == nil, c != nil && c.Enabled(), url == "")
		return nil
	}
	if c.IsCached(url) {
		debugLog.Printf("Fetch: skip already cached url=%s", truncateURL(url))
		return nil
	}
	if _, loaded := c.pending.LoadOrStore(url, true); loaded {
		debugLog.Printf("Fetch: skip already pending url=%s", truncateURL(url))
		return nil
	}

	debugLog.Printf("Fetch: starting download url=%s", truncateURL(url))

	return func() tea.Msg {
		defer c.pending.Delete(url)

		resp, err := httpClient.Get(url)
		if err != nil {
			debugLog.Printf("Fetch: HTTP error url=%s err=%v", truncateURL(url), err)
			c.setError(fmt.Errorf("fetch %s: %w", truncateURL(url), err))
			return nil
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if resp.StatusCode != http.StatusOK {
			debugLog.Printf("Fetch: HTTP status %d url=%s", resp.StatusCode, truncateURL(url))
			c.setError(fmt.Errorf("fetch %s: status %d", truncateURL(url), resp.StatusCode))
			return nil
		}

		img, format, err := image.Decode(resp.Body)
		if err != nil {
			debugLog.Printf("Fetch: decode error url=%s err=%v", truncateURL(url), err)
			c.setError(fmt.Errorf("decode %s: %w", truncateURL(url), err))
			return nil
		}

		debugLog.Printf("Fetch: decoded %s %dx%d url=%s", format, img.Bounds().Dx(), img.Bounds().Dy(), truncateURL(url))

		if transform != nil {
			img = transform(img)
			debugLog.Printf("Fetch: applied transform %dx%d url=%s", img.Bounds().Dx(), img.Bounds().Dy(), truncateURL(url))
		}

		hash := sha256.Sum256([]byte(url))
		filename := fmt.Sprintf("%x.png", hash[:8])
		path := filepath.Join(c.dir, filename)

		f, err := os.Create(path)
		if err != nil {
			debugLog.Printf("Fetch: create error path=%s err=%v", path, err)
			c.setError(fmt.Errorf("create %s: %w", path, err))
			return nil
		}
		defer func() {
			_ = f.Close()
		}()

		if err := png.Encode(f, img); err != nil {
			_ = os.Remove(path)
			debugLog.Printf("Fetch: encode error path=%s err=%v", path, err)
			c.setError(fmt.Errorf("encode %s: %w", path, err))
			return nil
		}

		debugLog.Printf("Fetch: saved path=%s url=%s", path, truncateURL(url))
		c.files.Store(url, path)

		// /dev/tty writes (Kitty APC) deferred to renderKittyPlaceholders (View goroutine)
		// to avoid interleaving with bubbletea's stdout writes from this background goroutine.

		return ImageFetchedMsg{URL: url}
	}
}

// RenderImage renders the cached image at the given size using the best
// available graphics protocol via go-termimg's Auto detection.
// Returns empty string if not cached, not enabled, or rendering fails.
func (c *Cache) RenderImage(url string, cols, rows int) string {
	if c == nil || !c.Enabled() {
		return ""
	}

	if _, ok := c.files.Load(url); !ok {
		return ""
	}

	if c.protocol == termimg.Kitty && c.tty != nil {
		return c.renderKittyPlaceholders(url, cols, rows)
	}

	return c.renderWidget(url, cols, rows)
}

func (c *Cache) renderKittyPlaceholders(url string, cols, rows int) string {
	numVal, ok := c.imageNums.Load(url)
	if !ok {
		// Lazy transmit: runs on main goroutine (View), safe from /dev/tty races
		pathVal, pathOk := c.files.Load(url)
		if !pathOk {
			return ""
		}
		imageNum, err := c.transmitImageData(pathVal.(string))
		if err != nil {
			c.setError(fmt.Errorf("transmit %s: %w", truncateURL(url), err))
			return ""
		}
		c.imageNums.Store(url, imageNum)
		numVal = imageNum
	}
	imageNum := numVal.(uint32)

	if info, ok := c.placements.Load(url); ok {
		pi := info.(placementInfo)
		if pi.cols != cols || pi.rows != rows {
			c.deleteVirtualPlacement(imageNum)
			c.setVirtualPlacement(imageNum, cols, rows)
			c.placements.Store(url, placementInfo{cols: cols, rows: rows, imageNum: imageNum})
		}
	} else {
		c.setVirtualPlacement(imageNum, cols, rows)
		c.placements.Store(url, placementInfo{cols: cols, rows: rows, imageNum: imageNum})
	}

	area := termimg.CreatePlaceholderArea(imageNum, uint16(rows), uint16(cols))
	output := termimg.RenderPlaceholderAreaWithImageID(area, imageNum)
	// Strip \r — termimg uses \r\n line endings but cellbuf only expects \n
	return strings.ReplaceAll(output, "\r", "")
}

func (c *Cache) renderWidget(url string, cols, rows int) string {
	pathVal, ok := c.files.Load(url)
	if !ok {
		return ""
	}
	path := pathVal.(string)

	if val, ok := c.widgets.Load(url); ok {
		w := val.(*termimg.ImageWidget)
		w.SetSize(cols, rows).SetProtocol(termimg.Auto)
		rendered, err := w.Render()
		if err != nil {
			c.setError(fmt.Errorf("render %s: %w", truncateURL(url), err))
			return ""
		}
		return rendered
	}

	widget, err := termimg.NewImageWidgetFromFile(path)
	if err != nil {
		c.setError(fmt.Errorf("widget %s: %w", path, err))
		return ""
	}

	widget.SetSize(cols, rows).SetProtocol(termimg.Auto)
	rendered, err := widget.Render()
	if err != nil {
		c.setError(fmt.Errorf("render %s: %w", truncateURL(url), err))
		return ""
	}

	c.widgets.Store(url, widget)
	return rendered
}

// DebugInfo returns diagnostic information about the cache state.
func (c *Cache) DebugInfo() string {
	if c == nil {
		return "cache=nil"
	}

	var cached, widgetCount, transmittedCount int
	c.files.Range(func(_, _ any) bool { cached++; return true })
	c.widgets.Range(func(_, _ any) bool { widgetCount++; return true })
	c.imageNums.Range(func(_, _ any) bool { transmittedCount++; return true })

	c.mu.Lock()
	errStr := "none"
	if c.lastErr != nil {
		errStr = c.lastErr.Error()
	}
	c.mu.Unlock()

	var detLog strings.Builder
	for _, d := range termimg.GetDetectionLog() {
		status := "ok"
		if !d.Success {
			status = "fail"
		}
		if d.Fallback {
			status += "(fallback)"
		}
		if d.Error != nil {
			_, _ = fmt.Fprintf(&detLog, " %s:%s(%v)", d.Protocol, status, d.Error)
		} else {
			_, _ = fmt.Fprintf(&detLog, " %s:%s", d.Protocol, status)
		}
	}

	return fmt.Sprintf("protocol=%s enabled=%v cached=%d widgets=%d transmitted=%d tty=%v dir=%s lastErr=%s detection=[%s]",
		c.protocol, c.Enabled(), cached, widgetCount, transmittedCount, c.tty != nil, c.dir, errStr, strings.TrimSpace(detLog.String()))
}

// transmitImageData uploads PNG data to the terminal via Kitty APC protocol.
// Returns the assigned image number for placeholder references.
func (c *Cache) transmitImageData(path string) (uint32, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	imageNum := atomic.AddUint32(&imageNumCounter, 1)
	encoded := base64.StdEncoding.EncodeToString(data)

	var buf strings.Builder
	const chunkSize = 4096
	first := true
	for len(encoded) > 0 {
		chunk := encoded
		more := 0
		if len(encoded) > chunkSize {
			chunk = encoded[:chunkSize]
			encoded = encoded[chunkSize:]
			more = 1
		} else {
			encoded = ""
		}

		if first {
			_, _ = fmt.Fprintf(&buf, "\x1b_Gf=100,t=d,i=%d,q=2,m=%d;%s\x1b\\", imageNum, more, chunk)
			first = false
		} else {
			_, _ = fmt.Fprintf(&buf, "\x1b_Gm=%d,q=2;%s\x1b\\", more, chunk)
		}
	}

	if _, err := c.tty.WriteString(buf.String()); err != nil {
		return 0, fmt.Errorf("tty write: %w", err)
	}

	debugLog.Printf("transmitImageData: sent imageNum=%d dataLen=%d path=%s", imageNum, len(data), path)
	return imageNum, nil
}

func (c *Cache) setVirtualPlacement(imageNum uint32, cols, rows int) {
	seq := fmt.Sprintf("\x1b_Ga=p,U=1,i=%d,c=%d,r=%d,q=2\x1b\\", imageNum, cols, rows)
	c.tty.WriteString(seq) //nolint:errcheck
}

func (c *Cache) deleteVirtualPlacement(imageNum uint32) {
	seq := fmt.Sprintf("\x1b_Ga=d,d=i,i=%d,q=2\x1b\\", imageNum)
	c.tty.WriteString(seq) //nolint:errcheck
}

func (c *Cache) Close() {
	if c == nil {
		return
	}
	if c.tty != nil {
		c.tty.WriteString("\x1b_Ga=d,d=A,q=2\x1b\\") //nolint:errcheck
		_ = c.tty.Close()
		c.tty = nil
	}
}

func (c *Cache) setError(err error) {
	c.mu.Lock()
	c.lastErr = err
	c.mu.Unlock()
}

func applyRoundedCorners(src image.Image, radiusPct float64, bgColor color.RGBA) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{bgColor}, image.Point{}, draw.Src)
	r := radiusPct * float64(min(w, h))

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if inRoundedRect(x, y, w, h, r) {
				dst.Set(x, y, src.At(x+b.Min.X, y+b.Min.Y))
			}
		}
	}
	return dst
}

func inRoundedRect(x, y, w, h int, r float64) bool {
	fx, fy := float64(x)+0.5, float64(y)+0.5

	var cx, cy float64
	if fx < r {
		cx = r
	} else if fx > float64(w)-r {
		cx = float64(w) - r
	} else {
		return true
	}

	if fy < r {
		cy = r
	} else if fy > float64(h)-r {
		cy = float64(h) - r
	} else {
		return true
	}

	dx := fx - cx
	dy := fy - cy
	return dx*dx+dy*dy <= r*r
}

func truncateURL(url string) string {
	if len(url) <= 60 {
		return url
	}
	return url[:57] + "..."
}
