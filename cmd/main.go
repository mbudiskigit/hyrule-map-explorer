package main

import (
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"math"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
)

type Game struct {
	bg *ebiten.Image
	// viewport in background image coordinates (top-left)
	vx, vy int
	// tile size in background pixels
	tileW, tileH int
	// player position in world coordinates (pixels)
	px, py float64
	// player sprite and size
	playerSprite     *ebiten.Image
	playerW, playerH int
}

func loadImage(path string) (*ebiten.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	return ebiten.NewImageFromImage(img), nil
}

func (g *Game) Update() error {
	// allow basic arrow-key panning between tiles
	// move tile indices when arrow keys are pressed
	const moveDelta = 1
	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		g.vx += moveDelta * g.tileW
	}
	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		g.vx -= moveDelta * g.tileW
	}
	if ebiten.IsKeyPressed(ebiten.KeyDown) {
		g.vy += moveDelta * g.tileH
	}
	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		g.vy -= moveDelta * g.tileH
	}
	// player movement with WASD keys
	const playerSpeed = 3.0
	if ebiten.IsKeyPressed(ebiten.KeyW) {
		g.py -= playerSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) {
		g.py += playerSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		g.px -= playerSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) {
		g.px += playerSpeed
	}
	// clamp player to image bounds
	if g.bg != nil {
		bw, bh := g.bg.Bounds().Dx(), g.bg.Bounds().Dy()
		if g.px < 0 {
			g.px = 0
		}
		if g.py < 0 {
			g.py = 0
		}
		if g.px > float64(bw-g.playerW) {
			g.px = float64(bw - g.playerW)
		}
		if g.py > float64(bh-g.playerH) {
			g.py = float64(bh - g.playerH)
		}
		// compute scale for current viewport
		vw := g.tileW
		vh := g.tileH
		// desired viewport center to match player center on screen
		desiredVx := int(g.px + float64(g.playerW)/2 - float64(vw)/2)
		desiredVy := int(g.py + float64(g.playerH)/2 - float64(vh)/2)
		// clamp viewport to image bounds
		if desiredVx < 0 {
			desiredVx = 0
		}
		if desiredVy < 0 {
			desiredVy = 0
		}
		if desiredVx > bw-g.tileW {
			desiredVx = bw - g.tileW
		}
		if desiredVy > bh-g.tileH {
			desiredVy = bh - g.tileH
		}
		g.vx = desiredVx
		g.vy = desiredVy
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.bg == nil {
		return
	}

	// screen size
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()

	// desired viewport in background image coordinates (use tileW/tileH)
	vw := g.tileW
	vh := g.tileH

	// compute scale to cover the screen while preserving aspect ratio
	sx := float64(sw) / float64(vw)
	sy := float64(sh) / float64(vh)
	// use the larger scale so the viewport covers the whole screen (no empty bars)
	scale := math.Max(sx, sy)

	op := &ebiten.DrawImageOptions{}
	// Scale first, then translate so that the viewport's (vx,vy) maps to
	// the screen origin, and then center the scaled viewport on the screen.
	// After scaling, add an offset to center if the scaled viewport is larger
	// than the screen in one dimension.
	dx := (float64(sw) - float64(vw)*scale) / 2
	dy := (float64(sh) - float64(vh)*scale) / 2
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(-float64(g.vx)*scale+dx, -float64(g.vy)*scale+dy)

	screen.DrawImage(g.bg, op)

	// draw player sprite
	// convert player world position to screen position
	playerScreenX := (g.px-float64(g.vx))*scale + dx
	playerScreenY := (g.py-float64(g.vy))*scale + dy

	// draw player sprite (already resized, scale for viewport)
	playerOp := &ebiten.DrawImageOptions{}
	playerOp.GeoM.Scale(scale, scale)
	playerOp.GeoM.Translate(playerScreenX, playerScreenY)
	if g.playerSprite != nil {
		screen.DrawImage(g.playerSprite, playerOp)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return outsideWidth, outsideHeight
}

func main() {
	// load background from assets folder
	imgPath := "assets/map-part1.jpg"
	bg, err := loadImage(imgPath)
	if err != nil {
		log.Fatalf("failed to load background image %s: %v", imgPath, err)
	}

	// derive tile size from the image by splitting it into a grid based on
	// a target tile size (in pixels). This calculates how many columns and
	// rows are needed so each tile is about `targetTile` pixels wide/tall.
	bw, bh := bg.Bounds().Dx(), bg.Bounds().Dy()
	const targetTile = 512
	cols := (bw + targetTile - 1) / targetTile // ceil(bw/targetTile)
	rows := (bh + targetTile - 1) / targetTile // ceil(bh/targetTile)
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}
	tileW := bw / cols
	tileH := bh / rows
	if tileW <= 0 {
		tileW = bw
	}
	if tileH <= 0 {
		tileH = bh
	}
	// set a larger default window size and allow resizing
	ebiten.SetWindowSize(1024, 768)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle("Hyrule Map Explorer")

	// calculate player size as 1% of the smallest screen dimension
	screenW, screenH := ebiten.WindowSize()
	minScreenDim := screenW
	if screenH < minScreenDim {
		minScreenDim = screenH
	}
	playerSize := int(float64(minScreenDim) * 0.03)
	if playerSize < 1 {
		playerSize = 1
	}
	playerW, playerH := playerSize, playerSize

	// load player sprite
	playerSpritePath := "assets/chest.png"
	playerSpriteOrig, err := loadImage(playerSpritePath)
	if err != nil {
		log.Fatalf("failed to load player sprite %s: %v", playerSpritePath, err)
	}

	// resize sprite to player size
	playerSprite := ebiten.NewImage(playerW, playerH)
	op := &ebiten.DrawImageOptions{}
	// calculate scale to fit sprite into playerW x playerH
	spriteW, spriteH := playerSpriteOrig.Bounds().Dx(), playerSpriteOrig.Bounds().Dy()
	scaleX := float64(playerW) / float64(spriteW)
	scaleY := float64(playerH) / float64(spriteH)
	op.GeoM.Scale(scaleX, scaleY)
	playerSprite.DrawImage(playerSpriteOrig, op)

	playerX := float64((tileW / 2) - (playerW / 2))
	playerY := float64((tileH / 2) - (playerH / 2))
	g := &Game{bg: bg, vx: 0, vy: 0, tileW: tileW, tileH: tileH, px: playerX, py: playerY, playerSprite: playerSprite, playerW: playerW, playerH: playerH}

	// start in fullscreen mode
	// ebiten.SetFullscreen(true)

	if err := ebiten.RunGame(g); err != nil {
		panic(err)
	}
}
