package d2player

import (
	"fmt"
	"log"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2ui"
)

const (
	frameInventoryTopLeft     = 4
	frameInventoryTopRight    = 5
	frameInventoryBottomLeft  = 6
	frameInventoryBottomRight = 7
)

const (
	invCloseButtonX, invCloseButtonY = 419, 449
)

// Inventory represents the inventory
type Inventory struct {
	asset       *d2asset.AssetManager
	item        *diablo2item.ItemFactory
	uiManager   *d2ui.UIManager
	frame       *d2ui.UIFrame
	panel       *d2ui.Sprite
	grid        *ItemGrid
	itemTooltip *d2ui.Tooltip
	closeButton *d2ui.Button
	hoverX      int
	hoverY      int
	originX     int
	originY     int
	lastMouseX  int
	lastMouseY  int
	hovering    bool
	isOpen      bool
	onCloseCb   func()
}

// NewInventory creates an inventory instance and returns a pointer to it
func NewInventory(asset *d2asset.AssetManager, ui *d2ui.UIManager,
	record *d2records.InventoryRecord) *Inventory {
	itemTooltip := ui.NewTooltip(d2resource.FontFormal11, d2resource.PaletteStatic, d2ui.TooltipXCenter, d2ui.TooltipYBottom)

	// https://github.com/OpenDiablo2/OpenDiablo2/issues/797
	itemFactory, _ := diablo2item.NewItemFactory(asset)

	return &Inventory{
		asset:       asset,
		uiManager:   ui,
		item:        itemFactory,
		grid:        NewItemGrid(asset, ui, record),
		originX:     record.Panel.Left,
		itemTooltip: itemTooltip,
		// originY: record.Panel.Top,
		originY: 0, // expansion data has these all offset by +60 ...
	}
}

// IsOpen returns true if the inventory is open
func (g *Inventory) IsOpen() bool {
	return g.isOpen
}

// Toggle negates the open state of the inventory
func (g *Inventory) Toggle() {
	if g.isOpen {
		g.Close()
	} else {
		g.Open()
	}
}

// Open opens the inventory
func (g *Inventory) Open() {
	g.isOpen = true
	g.closeButton.SetVisible(true)
}

// Close closes the inventory
func (g *Inventory) Close() {
	g.isOpen = false
	g.closeButton.SetVisible(false)
	g.onCloseCb()
}

// SetOnCloseCb the callback run on closing the inventory
func (g *Inventory) SetOnCloseCb(cb func()) {
	g.onCloseCb = cb
}

// Load the resources required by the inventory
func (g *Inventory) Load() {
	g.frame = d2ui.NewUIFrame(g.asset, g.uiManager, d2ui.FrameRight)

	g.closeButton = g.uiManager.NewButton(d2ui.ButtonTypeSquareClose, "")
	g.closeButton.SetVisible(false)
	g.closeButton.SetPosition(invCloseButtonX, invCloseButtonY)
	g.closeButton.OnActivated(func() { g.Close() })

	g.panel, _ = g.uiManager.NewSprite(d2resource.InventoryCharacterPanel, d2resource.PaletteSky)

	// https://github.com/OpenDiablo2/OpenDiablo2/issues/795
	testInventoryCodes := [][]string{
		{"kit", "Crimson", "of the Bat", "of Frost"},
		{"rin", "Steel", "of Shock"},
		{"jav"},
		{"buc"},
	}

	inventoryItems := make([]InventoryItem, 0)

	for idx := range testInventoryCodes {
		item, err := g.item.NewItem(testInventoryCodes[idx]...)
		if err != nil {
			continue
		}

		item.Identify()
		inventoryItems = append(inventoryItems, item)
	}

	// https://github.com/OpenDiablo2/OpenDiablo2/issues/795
	testEquippedItemCodes := map[d2enum.EquippedSlot][]string{
		d2enum.EquippedSlotLeftArm:   {"wnd"},
		d2enum.EquippedSlotRightArm:  {"buc"},
		d2enum.EquippedSlotHead:      {"crn"},
		d2enum.EquippedSlotTorso:     {"plt"},
		d2enum.EquippedSlotLegs:      {"vbt"},
		d2enum.EquippedSlotBelt:      {"vbl"},
		d2enum.EquippedSlotGloves:    {"lgl"},
		d2enum.EquippedSlotLeftHand:  {"rin"},
		d2enum.EquippedSlotRightHand: {"rin"},
		d2enum.EquippedSlotNeck:      {"amu"},
	}

	for slot := range testEquippedItemCodes {
		item, err := g.item.NewItem(testEquippedItemCodes[slot]...)
		if err != nil {
			continue
		}

		g.grid.ChangeEquippedSlot(slot, item)
	}

	_, err := g.grid.Add(inventoryItems...)
	if err != nil {
		fmt.Printf("could not add items to the inventory, err: %v\n", err)
	}
}

// Render draws the inventory onto the given surface
func (g *Inventory) Render(target d2interface.Surface) {
	if !g.isOpen {
		return
	}

	err := g.renderFrame(target)
	if err != nil {
		log.Println(err)
	}

	g.grid.Render(target)
	g.renderItemHover(target)
}

func (g *Inventory) renderFrame(target d2interface.Surface) error {
	if err := g.frame.Render(target); err != nil {
		return err
	}

	frames := []int{
		frameInventoryTopLeft,
		frameInventoryTopRight,
		frameInventoryBottomRight,
		frameInventoryBottomLeft,
	}

	x, y := g.originX+1, g.originY
	y += 64

	for _, frame := range frames {
		if err := g.panel.SetCurrentFrame(frame); err != nil {
			return err
		}

		w, h := g.panel.GetCurrentFrameSize()

		g.panel.SetPosition(x, y+h)
		g.panel.Render(target)

		switch frame {
		case frameInventoryTopLeft:
			x += w
		case frameInventoryTopRight:
			y += h
		case frameInventoryBottomRight:
			x = g.originX + 1
		}
	}

	return nil
}

func (g *Inventory) renderItemHover(target d2interface.Surface) {
	hovering := false

	for idx := range g.grid.items {
		item := g.grid.items[idx]
		ix, iy := g.grid.SlotToScreen(item.InventoryGridSlot())
		iw, ih := g.grid.sprites[item.GetItemCode()].GetCurrentFrameSize()
		mx, my := g.lastMouseX, g.lastMouseY
		hovering = hovering || ((mx > ix) && (mx < ix+iw) && (my > iy) && (my < iy+ih))

		if hovering {
			if !g.hovering {
				// set the initial hover coordinates
				// this is so that moving mouse doesnt move the description
				g.hoverX, g.hoverY = mx, my
			}

			g.renderItemDescription(target, item)

			break
		}
	}

	g.hovering = hovering
}

func (g *Inventory) renderItemDescription(target d2interface.Surface, i InventoryItem) {
	lines := i.GetItemDescription()
	g.itemTooltip.SetTextLines(lines)
	_, y := g.grid.SlotToScreen(i.InventoryGridSlot())
	g.itemTooltip.SetPosition(g.hoverX, y)
	g.itemTooltip.Render(target)
}
