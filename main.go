package main

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

const NUM_TRAYS = 26

var VALUES = []int{
	1, 5, 10, 25, 50, 75, 100, 200,
	300, 400, 500, 750, 1000, 5000,
	10000, 12500, 25000, 50000, 75000,
	100000, 200000, 300000, 400000,
	500000, 750000, 1000000,
}

// Game holds the state and UI elements
type Game struct {
	win              fyne.Window
	gridButtons      []*widget.Button
	leftLabels       []*widget.Label
	rightLabels      []*widget.Label
	trayValues       []int
	trayReplaced     []int    // if tray had an item, stores the numeric value removed
	itemNames        []string // "" if none
	itemImages       []int    // food cartoon image IDs for items
	playerTray       int
	playerTrayButton *widget.Button // visual representation of player's tray
	openedTraysCount int
	openedValues     map[int]bool
	chef             *Chef
	bonus            *BonusManager
	bonusOffered     bool // track if bonus has been offered this game
}

func NewGame() *Game {
	return &Game{
		playerTray:   -1,
		chef:         NewChef(),
		bonus:        NewBonusManager(),
		openedValues: make(map[int]bool),
		bonusOffered: false,
	}
}

// Helper function to load image from file with better error handling
func loadImageSafe(filename string, width, height float32) fyne.CanvasObject {
	uri := storage.NewFileURI("images/" + filename)
	img := canvas.NewImageFromURI(uri)

	// Check if image loaded successfully
	if img == nil {
		// Return placeholder if image fails to load
		return widget.NewLabel(fmt.Sprintf("Image: %s", filename))
	}

	img.FillMode = canvas.ImageFillContain
	img.SetMinSize(fyne.NewSize(width, height))
	return img
}

// Update the old loadImage function to use the safe version
func loadImage(filename string, width, height float32) *canvas.Image {
	uri := storage.NewFileURI("images/" + filename)
	img := canvas.NewImageFromURI(uri)
	img.FillMode = canvas.ImageFillContain
	img.SetMinSize(fyne.NewSize(width, height))
	return img
}

// Helper function to get the closest value image for an offer
func (g *Game) getOfferImageID(offer int) int {
	// Find the closest value in VALUES array
	closestIdx := 0
	minDiff := abs(VALUES[0] - offer)

	for i := 1; i < len(VALUES); i++ {
		diff := abs(VALUES[i] - offer)
		if diff < minDiff {
			minDiff = diff
			closestIdx = i
		}
	}

	return closestIdx + 1 // Return 1-based index for image naming
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func (g *Game) initialize() {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// shuffle VALUES and assign one-to-one
	sh := make([]int, len(VALUES))
	copy(sh, VALUES)
	r.Shuffle(len(sh), func(i, j int) { sh[i], sh[j] = sh[j], sh[i] })

	g.trayValues = make([]int, NUM_TRAYS)
	for i := 0; i < NUM_TRAYS; i++ {
		g.trayValues[i] = sh[i%len(sh)]
	}

	// init arrays
	g.itemNames = make([]string, NUM_TRAYS)
	g.itemImages = make([]int, NUM_TRAYS)
	g.trayReplaced = make([]int, NUM_TRAYS)
	for i := range g.trayReplaced {
		g.trayReplaced[i] = -1
	}

	// choose 0..3 item replacements (avoid first/last)
	numItems := r.Intn(4)
	replace := map[int]bool{}
	for len(replace) < numItems {
		idx := r.Intn(NUM_TRAYS)
		// Skip if this tray contains lowest or highest value
		if g.trayValues[idx] == VALUES[0] || g.trayValues[idx] == VALUES[len(VALUES)-1] {
			continue
		}
		if !replace[idx] {
			replace[idx] = true
		}
	}

	// Food-themed items pool (cartoon food images: 51-100)
	foodItems := []struct {
		name    string
		imageID int
	}{
		{"Beigners", 51},
		{"Cheese Sandwich", 52},
		{"Magic cookies", 53},
		{"Ultimate sandwich", 54},
		{"Pretty patty", 55},
		{"hors d'oeuvres", 56},
		{"Nacco", 57},
		{"Krabby patty", 58},
		{"jr. patty", 59},
		{"Poritage", 60},
		{"Hot Dog", 61},
		{"Ramen", 62},
		{"Chilli fries", 63},
		{"Ultimate Sandwich", 64},
		{"Spanish puffs", 65},
		{"Turkey", 66},
		{"Dreamy breakfast", 67},
		{"ratatouille", 68},
	}

	for idx := range replace {
		food := foodItems[r.Intn(len(foodItems))]
		g.itemNames[idx] = food.name
		g.itemImages[idx] = food.imageID
		g.trayReplaced[idx] = g.trayValues[idx]
		g.trayValues[idx] = -1 // mark as item
	}

	// Build display sidebar in VALUES order; mark removed numeric values as FOOD ITEM sentinel
	display := make([]int, len(VALUES))
	copy(display, VALUES)

	removed := map[int]bool{}
	for _, v := range g.trayReplaced {
		if v != -1 {
			removed[v] = true
		}
	}
	for i := 0; i < len(display); i++ {
		if removed[display[i]] {
			// sentinel
			display[i] = -999999
		}
	}

	half := len(VALUES) / 2
	g.leftLabels = make([]*widget.Label, half)
	g.rightLabels = make([]*widget.Label, half)

	for i := 0; i < half; i++ {
		ltext := fmt.Sprintf("$%d", display[i])
		if display[i] == -999999 {
			ltext = "🍔 FOOD ITEM"
		}
		g.leftLabels[i] = widget.NewLabel(ltext)

		rtext := fmt.Sprintf("$%d", display[i+half])
		if display[i+half] == -999999 {
			rtext = "🍔 FOOD ITEM"
		}
		g.rightLabels[i] = widget.NewLabel(rtext)
	}
}

func (g *Game) setupUI(a fyne.App) fyne.CanvasObject {
	left := container.NewVBox()
	right := container.NewVBox()
	for i := 0; i < len(g.leftLabels); i++ {
		l := g.leftLabels[i]
		// Put each label into a small card (white box) for readability
		left.Add(widget.NewCard("", "", l))
	}
	for i := 0; i < len(g.rightLabels); i++ {
		l := g.rightLabels[i]
		right.Add(widget.NewCard("", "", l))
	}

	g.gridButtons = make([]*widget.Button, NUM_TRAYS)
	grid := container.NewGridWithColumns(6)
	for i := 0; i < NUM_TRAYS; i++ {
		index := i
		btn := widget.NewButton(fmt.Sprintf("🍽️ %d", i+1), func() {
			g.onTrayClicked(a, index)
		})
		btn.Importance = widget.HighImportance
		g.gridButtons[i] = btn
		grid.Add(btn)
	}

	center := container.NewHBox(left, grid, right)
	return center
}

func (g *Game) onTrayClicked(a fyne.App, idx int) {
	w := g.win

	// First pick → player's tray
	if g.playerTray == -1 {
		g.playerTray = idx

		// Create a visual representation of player's tray (same size as other trays)
		g.playerTrayButton = widget.NewButton(fmt.Sprintf("🍽️ %d", idx+1), nil)
		g.playerTrayButton.Importance = widget.HighImportance

		// IMPORTANT: Disable the button BEFORE rebuilding UI
		g.gridButtons[idx].Disable()

		// bottom indicator with the tray button
		bottom := container.NewCenter(
			container.NewHBox(
				widget.NewLabel("My Tray: "),
				g.playerTrayButton,
			),
		)
		content := g.setupUI(a)
		// Make sure the disabled state persists
		g.gridButtons[idx].Disable()

		w.SetContent(container.NewBorder(
			container.NewCenter(widget.NewLabel("🍽️ Meal or No Meal 🍽️")),
			bottom,
			nil,
			nil,
			container.NewCenter(content),
		))

		dialog.ShowInformation("Your Tray",
			fmt.Sprintf("You chose Tray %d. This is your tray until the end!", idx+1), w)
		return
	}

	// prevent re-opening player's tray
	if idx == g.playerTray {
		dialog.ShowInformation("Not Allowed", "That's your tray! You can't open it yet.", w)
		return
	}

	// open chosen tray
	g.gridButtons[idx].Disable()
	g.openedTraysCount++

	// Show tray opened dialog with image
	g.showTrayOpenedDialog(w, idx)
}

func (g *Game) showTrayOpenedDialog(parent fyne.Window, idx int) {
	var contentWidget fyne.CanvasObject

	if g.itemNames[idx] != "" {
		// Show food item with cartoon image
		foodImg := loadImage(fmt.Sprintf("%d.jpg", g.itemImages[idx]), 200, 200)
		label := widget.NewLabel(fmt.Sprintf("🍽️ Tray %d contains:\n%s", idx+1, g.itemNames[idx]))
		contentWidget = container.NewVBox(
			container.NewCenter(foodImg),
			container.NewCenter(label),
		)
	} else {
		// Show money value with corresponding image (1-26)
		// Find which position this value is in the VALUES array
		valueIndex := -1
		for i, v := range VALUES {
			if v == g.trayValues[idx] {
				valueIndex = i + 1 // 1-based for image naming
				break
			}
		}

		if valueIndex > 0 {
			moneyImg := loadImage(fmt.Sprintf("%d.jpg", valueIndex), 200, 200)
			label := widget.NewLabel(fmt.Sprintf("🍽️ Tray %d contains:\n$%d", idx+1, g.trayValues[idx]))
			contentWidget = container.NewVBox(
				container.NewCenter(moneyImg),
				container.NewCenter(label),
			)
		} else {
			// Fallback if image not found
			contentWidget = widget.NewLabel(fmt.Sprintf("🍽️ Tray %d contains:\n$%d", idx+1, g.trayValues[idx]))
		}
	}

	d := dialog.NewCustom("Tray Opened", "OK", contentWidget, parent)
	d.SetOnClosed(func() {
		// mark sidebar
		g.markPriceAsOpened(idx)

		// Check if it's time for chef offer (every 3 trays)
		if g.openedTraysCount%3 == 0 {
			g.showChefOffer(parent)
		} else if g.getUnopenedCount() == 1 {
			// final reveal when 1 left (not including player's tray)
			g.showFinalReveal(parent)
		}
	})
	d.Show()
}

func (g *Game) markPriceAsOpened(trayIndex int) {
	// If this tray originally had an item, mark the removed numeric value slot
	if g.trayReplaced[trayIndex] != -1 {
		removed := g.trayReplaced[trayIndex]
		g.openedValues[removed] = true // track it
		for i := 0; i < len(VALUES); i++ {
			if VALUES[i] == removed {
				if i < len(VALUES)/2 {
					lbl := g.leftLabels[i]
					lbl.SetText("✓ " + lbl.Text)
				} else {
					lbl := g.rightLabels[i-len(VALUES)/2]
					lbl.SetText("✓ " + lbl.Text)
				}
				return
			}
		}
	}

	// Otherwise find the matching numeric label and mark it
	val := g.trayValues[trayIndex]
	g.openedValues[val] = true // track it
	for i := 0; i < len(VALUES); i++ {
		if VALUES[i] == val {
			if i < len(VALUES)/2 {
				lbl := g.leftLabels[i]
				lbl.SetText("✓ " + lbl.Text)
			} else {
				lbl := g.rightLabels[i-len(VALUES)/2]
				lbl.SetText("✓ " + lbl.Text)
			}
			return
		}
	}
}

func (g *Game) getUnopenedCount() int {
	count := 0
	for i := 0; i < NUM_TRAYS; i++ {
		if i == g.playerTray {
			continue // Don't count player's tray
		}
		if !g.gridButtons[i].Disabled() {
			count++
		}
	}
	return count
}

func (g *Game) showChefOffer(parent fyne.Window) {
	// Check if final reveal
	if g.getUnopenedCount() == 1 {
		g.showFinalReveal(parent)
		return
	}
	remaining := []int{}
	// Include player's tray in remaining values
	for i := 0; i < NUM_TRAYS; i++ {
		// skip opened (disabled) trays, but include player's tray
		if i != g.playerTray && g.gridButtons[i].Disabled() {
			continue
		}
		// only include numeric values (skip items)
		if g.trayValues[i] != -1 {
			remaining = append(remaining, g.trayValues[i])
		}
	}
	if len(remaining) == 0 {
		return
	}

	// Trigger bonuses ONLY ONCE per game at a random chef offer
	// 30% chance to trigger bonus if not already offered
	if !g.bonusOffered && g.chef.r.Float64() < 0.30 {
		g.bonusOffered = true
		// Show bonuses BEFORE chef offer
		g.showBonusSequence(parent, remaining)
		return
	}

	// Normal chef offer flow (no bonus)
	if g.chef.OfferSwap() {
		// Create buttons with symbols
		acceptBtn := widget.NewButton("✓ Accept", nil)
		declineBtn := widget.NewButton("✗ Decline", nil)

		// Set colors: Accept = Blue, Decline = Grey
		acceptBtn.Importance = widget.HighImportance    // Blue
		declineBtn.Importance = widget.MediumImportance // Grey

		content := widget.NewLabel("🍽️ The Banker offers to swap your tray with another unopened one. Swap?")
		// Accept on LEFT, Decline on RIGHT
		buttons := container.NewHBox(acceptBtn, declineBtn)
		dialogContent := container.NewVBox(content, buttons)

		dlg := dialog.NewCustomWithoutButtons("Banker's Offer", dialogContent, parent)

		// Accept button = do the swap
		acceptBtn.OnTapped = func() {
			dlg.Hide()
			g.swapTray(parent)
		}

		// Decline button = don't swap
		declineBtn.OnTapped = func() {
			dlg.Hide()
		}

		dlg.Show()
		return
	}

	offer := g.chef.CalculateOffer(remaining)

	// Show the offer dialog
	g.showOfferDialog(parent, offer)

}

// Show bonuses in sequence BEFORE chef offer
func (g *Game) showBonusSequence(parent fyne.Window, remaining []int) {
	hasMultiplier := g.bonus.HasMultiplier()
	hasAdditive := g.bonus.HasAdditive()

	if hasMultiplier && hasAdditive {
		// Show multiplier first, then additive, then chef offer
		g.bonus.TriggerMultiplierWithCallback(parent, func() {
			g.bonus.TriggerAdditiveWithCallback(parent, func() {
				// After both bonuses, proceed with chef offer
				g.proceedWithChefOffer(parent, remaining)
			})
		})
	} else if hasMultiplier {
		// Show multiplier, then chef offer
		g.bonus.TriggerMultiplierWithCallback(parent, func() {
			g.proceedWithChefOffer(parent, remaining)
		})
	} else if hasAdditive {
		// Show additive, then chef offer
		g.bonus.TriggerAdditiveWithCallback(parent, func() {
			g.proceedWithChefOffer(parent, remaining)
		})
	} else {
		// No bonuses available, proceed directly
		g.proceedWithChefOffer(parent, remaining)
	}
}

func (g *Game) proceedWithChefOffer(parent fyne.Window, remaining []int) {
	// If chef proposes swap
	if g.chef.OfferSwap() {
		//g.showSwapOfferDialog(parent)
		return
	}

	offer := g.chef.CalculateOffer(remaining)

	// Show bonus value being applied if there is one
	if g.bonus.HasPendingBonus() {
		originalOffer := offer
		offer = g.bonus.Apply(offer)
		bonusMsg := g.bonus.GetBonusDescription()

		// Only show bonus dialog if there's actually a bonus description
		if bonusMsg != "" {
			// Show both original and new offer images
			originalImgID := g.getOfferImageID(originalOffer)
			newImgID := g.getOfferImageID(offer)

			originalImg := loadImage(fmt.Sprintf("%d.jpg", originalImgID), 120, 120)
			newImg := loadImage(fmt.Sprintf("%d.jpg", newImgID), 120, 120)

			bonusContent := container.NewVBox(
				widget.NewLabel(" Bonus Applied!"),
				widget.NewLabel(bonusMsg),
				widget.NewSeparator(),
				container.NewHBox(
					container.NewVBox(
						widget.NewLabel("Original Offer:"),
						container.NewCenter(originalImg),
						widget.NewLabel(fmt.Sprintf("$%d", originalOffer)),
					),
					widget.NewLabel("  →  "),
					container.NewVBox(
						widget.NewLabel("New Offer:"),
						container.NewCenter(newImg),
						widget.NewLabel(fmt.Sprintf("$%d", offer)),
					),
				),
			)

			d := dialog.NewCustom("Bonus Applied!", "Continue", bonusContent, parent)
			d.SetOnClosed(func() {
				g.showOfferDialog(parent, offer)
			})
			d.Show()
		} else {
			// No bonus description, skip directly to offer
			g.showOfferDialog(parent, offer)
		}
	} else {
		g.showOfferDialog(parent, offer)
	}
}

// Helper function to show offer dialog with custom buttons and chef image
func (g *Game) showOfferDialog(parent fyne.Window, offer int) {
	// Get random chef image
	chefImgID := g.chef.GetRandomChefImage()
	chefImg := loadImage(fmt.Sprintf("%d.jpg", chefImgID), 200, 200)

	// Create buttons with symbols
	acceptBtn := widget.NewButton("✓ Accept", nil)
	declineBtn := widget.NewButton("✗ Decline", nil)

	// Set colors: Accept = Blue, Decline = Grey
	acceptBtn.Importance = widget.HighImportance    // Blue
	declineBtn.Importance = widget.MediumImportance // Grey

	content := widget.NewLabel(fmt.Sprintf("‍ The Chef offers you: $%d\nMeal or No Meal?", offer))
	// Accept on LEFT, Decline on RIGHT
	buttons := container.NewHBox(acceptBtn, declineBtn)

	dialogContent := container.NewVBox(
		container.NewCenter(chefImg),
		widget.NewSeparator(),
		content,
		buttons,
	)

	dlg := dialog.NewCustomWithoutButtons("Chef's Offer", dialogContent, parent)

	// Accept button = take the deal
	acceptBtn.OnTapped = func() {
		dlg.Hide()
		g.showDealAccepted(parent, offer)
	}

	// Decline button = continue playing
	declineBtn.OnTapped = func() {
		dlg.Hide()
	}

	dlg.Show()
}

func (g *Game) swapTray(parent fyne.Window) {
	// build list of available unopened trays
	options := []string{}
	for i := 0; i < NUM_TRAYS; i++ {
		if i != g.playerTray && !g.gridButtons[i].Disabled() {
			options = append(options, fmt.Sprintf("%d", i+1))
		}
	}
	if len(options) == 0 {
		dialog.ShowInformation("Swap", "No unopened trays available to swap.", parent)
		return
	}

	selectW := widget.NewSelect(options, func(s string) {})
	selectW.PlaceHolder = "Choose tray number"

	swapBtn := widget.NewButton("🔄 Swap", nil)
	swapBtn.Importance = widget.HighImportance // Blue button

	dialogContent := container.NewVBox(
		widget.NewLabel("Choose a tray to swap with:"),
		selectW,
		container.NewCenter(swapBtn), // Center the button
	)

	dlg := dialog.NewCustomWithoutButtons("Swap Tray", dialogContent, parent)

	swapBtn.OnTapped = func() {
		if selectW.Selected != "" {
			chosen, _ := strconv.Atoi(selectW.Selected)
			newIdx := chosen - 1
			oldPlayerTray := g.playerTray

			// swap trayValues, itemNames, trayReplaced
			g.trayValues[oldPlayerTray], g.trayValues[newIdx] = g.trayValues[newIdx], g.trayValues[oldPlayerTray]
			g.itemNames[oldPlayerTray], g.itemNames[newIdx] = g.itemNames[newIdx], g.itemNames[oldPlayerTray]
			g.trayReplaced[oldPlayerTray], g.trayReplaced[newIdx] = g.trayReplaced[newIdx], g.trayReplaced[oldPlayerTray]

			// Update player tray to new index
			g.playerTray = newIdx

			// Enable old tray in grid, disable new tray in grid
			g.gridButtons[oldPlayerTray].Enable()
			g.gridButtons[newIdx].Disable()

			// Update player tray button display on the right
			g.playerTrayButton.SetText(fmt.Sprintf("🍽️ %d", newIdx+1))

			g.refreshLabels()

			dlg.Hide()
			dialog.ShowInformation("Swap Completed",
				fmt.Sprintf("You swapped to Tray %d", g.playerTray+1), parent)
		}
	}

	dlg.Show()
}

func (g *Game) refreshLabels() {
	half := len(VALUES) / 2

	// Rebuild the display array to properly handle "FOOD ITEM" markers
	display := make([]int, len(VALUES))
	copy(display, VALUES)

	// Mark replaced values with sentinel
	removed := map[int]bool{}
	for _, v := range g.trayReplaced {
		if v != -1 {
			removed[v] = true
		}
	}
	for i := 0; i < len(display); i++ {
		if removed[display[i]] {
			display[i] = -999999 // sentinel for FOOD ITEM
		}
	}

	for i := 0; i < half; i++ {
		valLeft := display[i]
		var textLeft string
		if valLeft == -999999 {
			textLeft = "🍽️ FOOD ITEM"
		} else {
			textLeft = fmt.Sprintf("$%d", valLeft)
		}
		if g.isValueOpened(VALUES[i]) { // check against original VALUES
			textLeft = "✓ " + textLeft
		}
		g.leftLabels[i].SetText(textLeft)

		valRight := display[i+half]
		var textRight string
		if valRight == -999999 {
			textRight = "🍽️ FOOD ITEM"
		} else {
			textRight = fmt.Sprintf("$%d", valRight)
		}
		if g.isValueOpened(VALUES[i+half]) { // check against original VALUES
			textRight = "✓ " + textRight
		}
		g.rightLabels[i].SetText(textRight)
	}
}

func (g *Game) isValueOpened(val int) bool {
	if g.openedValues == nil {
		return false
	}
	return g.openedValues[val]
}

func (g *Game) showFinalReveal(parent fyne.Window) {
	var contentWidget fyne.CanvasObject
	if g.itemNames[g.playerTray] != "" {
		foodImg := loadImage(fmt.Sprintf("%d.jpg", g.itemImages[g.playerTray]), 200, 200)
		contentWidget = container.NewVBox(
			widget.NewLabel(fmt.Sprintf("🍽️ Your tray (Tray %d) contains: %s", g.playerTray+1, g.itemNames[g.playerTray])),
			widget.NewSeparator(),
			container.NewCenter(foodImg),
		)

	} else {
		// Show money value with image
		valueIndex := -1
		for i, v := range VALUES {
			if v == g.trayValues[g.playerTray] {
				valueIndex = i + 1
				break
			}
		}

		if valueIndex > 0 {
			moneyImg := loadImage(fmt.Sprintf("%d.jpg", valueIndex), 200, 200)
			label := widget.NewLabel(fmt.Sprintf("Your tray (Tray %d) contained:\n$%d", g.playerTray+1, g.trayValues[g.playerTray]))
			contentWidget = container.NewVBox(
				widget.NewSeparator(),
				container.NewCenter(label),
				container.NewCenter(moneyImg),
			)
		}
	}

	d := dialog.NewCustom("Final Reveal", "OK", contentWidget, parent)
	d.SetOnClosed(func() {
		for _, b := range g.gridButtons {
			b.Disable()
		}
		g.showPlayAgain(parent)
	})
	d.Show()
}

func (g *Game) showPlayAgain(parent fyne.Window) {
	playAgainBtn := widget.NewButton("🔄 Play Again", func() {
		// Start a fresh game
		ng := NewGame()
		ng.win = parent
		ng.initialize()
		content := ng.setupUI(fyne.CurrentApp())
		parent.SetContent(container.NewBorder(
			container.NewCenter(widget.NewLabel("🍽️ Meal or No Meal 🍽️")),
			nil,
			nil,
			nil,
			container.NewCenter(content),
		))
	})

	closeBtn := widget.NewButton("❌ Close", func() {
		// Close the window and quit the app
		parent.Close()
		fyne.CurrentApp().Quit()
	})

	// Create buttons container
	buttonsContainer := container.NewHBox(playAgainBtn, closeBtn)

	// Create and show dialog, store reference so we can hide it
	dlg := dialog.NewCustomWithoutButtons("🎮 Game Over", buttonsContainer, parent)

	// Update play again button to hide dialog first
	playAgainBtn.OnTapped = func() {
		dlg.Hide()
		// Start a fresh game
		ng := NewGame()
		ng.win = parent
		ng.initialize()
		content := ng.setupUI(fyne.CurrentApp())
		parent.SetContent(container.NewBorder(
			container.NewCenter(widget.NewLabel("🍽️ Meal or No Meal 🍽️")),
			nil,
			nil,
			nil,
			container.NewCenter(content),
		))
	}

	closeBtn.OnTapped = func() {
		dlg.Hide()
		parent.Close()
		fyne.CurrentApp().Quit()
	}

	dlg.Show()
}

func (g *Game) showDealAccepted(parent fyne.Window, offer int) {
	// Get random chef image for the accepted deal
	chefImgID := g.chef.GetRandomChefImage()
	chefImg := loadImage(fmt.Sprintf("%d.jpg", chefImgID), 200, 200)

	var contentWidget fyne.CanvasObject

	if g.itemNames[g.playerTray] != "" {
		// Show food item with image
		foodImg := loadImage(fmt.Sprintf("%d.jpg", g.itemImages[g.playerTray]), 200, 200)
		label := widget.NewLabel(fmt.Sprintf(" Your tray (Tray %d) contained:\n%s", g.playerTray+1, g.itemNames[g.playerTray]))
		contentWidget = container.NewVBox(
			widget.NewLabel(fmt.Sprintf("You accepted the deal!\n️  Your reward:  %d", offer)),
			container.NewCenter(chefImg),
			widget.NewSeparator(),
			container.NewCenter(label),
			container.NewCenter(foodImg),
		)
	} else {
		// Show money value with image
		valueIndex := -1
		for i, v := range VALUES {
			if v == g.trayValues[g.playerTray] {
				valueIndex = i + 1
				break
			}
		}

		if valueIndex > 0 {
			moneyImg := loadImage(fmt.Sprintf("%d.jpg", valueIndex), 200, 200)
			label := widget.NewLabel(fmt.Sprintf("Your tray (Tray %d) contained:\n$%d", g.playerTray+1, g.trayValues[g.playerTray]))
			contentWidget = container.NewVBox(
				widget.NewLabel(fmt.Sprintf("You accepted the deal!\n️  Your reward:  %d", offer)),
				container.NewCenter(chefImg),
				widget.NewSeparator(),
				container.NewCenter(label),
				container.NewCenter(moneyImg),
			)
		} else {
			contentWidget = container.NewVBox(
				widget.NewLabel(fmt.Sprintf("You accepted the deal!\n️  Your reward:  %d", offer)),
				container.NewCenter(chefImg),
				widget.NewSeparator(),
				widget.NewLabel(fmt.Sprintf("Your tray (Tray %d) contained:\n$%d", g.playerTray+1, g.trayValues[g.playerTray])),
			)
		}
	}

	d := dialog.NewCustom("Game Over - Deal Accepted!", "OK", contentWidget, parent)
	d.SetOnClosed(func() {
		for _, b := range g.gridButtons {
			b.Disable()
		}
		g.showPlayAgain(parent)
	})
	d.Show()
}

func main() {
	a := app.New()
	w := a.NewWindow("🍽️ Meal or No Meal 🍽️")
	g := NewGame()
	g.win = w
	g.initialize()

	content := g.setupUI(a)
	w.SetContent(container.NewBorder(
		container.NewCenter(widget.NewLabel("🍽️ Meal or No Meal 🍽️")),
		nil,
		nil,
		nil,
		container.NewCenter(content),
	))
	w.Resize(fyne.NewSize(1000, 600))
	w.ShowAndRun()
}
