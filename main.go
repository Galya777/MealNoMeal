package main

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
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
	playerTray       int
	playerTrayButton *widget.Button // visual representation of player's tray
	openedTraysCount int
	openedValues     map[int]bool
	banker           *Banker
	bonus            *BonusManager
	bonusOffered     bool // track if bonus has been offered this game
}

func NewGame() *Game {
	return &Game{
		playerTray:   -1,
		banker:       NewBanker(),
		bonus:        NewBonusManager(),
		openedValues: make(map[int]bool),
		bonusOffered: false,
	}
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
	g.trayReplaced = make([]int, NUM_TRAYS)
	for i := range g.trayReplaced {
		g.trayReplaced[i] = -1
	}

	// choose 0..3 item replacements (avoid first/last)
	numItems := r.Intn(4)
	replace := map[int]bool{}
	for len(replace) < numItems {
		idx := r.Intn(NUM_TRAYS)
		if idx == 0 || idx == NUM_TRAYS-1 {
			continue
		}
		if !replace[idx] {
			replace[idx] = true
		}
	}

	// Food-themed items pool
	pool := []string{
		"🍕 Pizza", "🍔 Burger", "🍟 Fries", "🌮 Taco", "🍝 Pasta", "🍣 Sushi",
		"🍰 Cake", "🍦 Ice Cream", "🥗 Salad", "🍗 Chicken", "🥩 Steak", "🍤 Shrimp",
		"🌭 Hot Dog", "🥪 Sandwich", "🍜 Ramen", "🍱 Bento Box", "🧀 Cheese Platter", "🥘 Paella",
	}

	for idx := range replace {
		name := pool[r.Intn(len(pool))]
		g.itemNames[idx] = name
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
			ltext = "🍽️ FOOD ITEM"
		}
		g.leftLabels[i] = widget.NewLabel(ltext)

		rtext := fmt.Sprintf("$%d", display[i+half])
		if display[i+half] == -999999 {
			rtext = "🍽️ FOOD ITEM"
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

	var contentText string
	if g.itemNames[idx] != "" {
		contentText = fmt.Sprintf("🍽️ You opened Tray %d\nIt contained: %s", idx+1, g.itemNames[idx])
	} else {
		contentText = fmt.Sprintf("🍽️ You opened Tray %d\nIt contained: $%d", idx+1, g.trayValues[idx])
	}

	// Show tray opened dialog, then proceed with banker offer in callback
	d := dialog.NewInformation("Tray Opened", contentText, w)
	d.SetOnClosed(func() {
		// mark sidebar
		g.markPriceAsOpened(idx)

		// Check if it's time for banker offer (every 3 trays)
		if g.openedTraysCount%3 == 0 {
			g.showBankerOffer(w)
		} else if g.getUnopenedCount() == 1 {
			// final reveal when 1 left (not including player's tray)
			g.showFinalReveal(w)
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

func (g *Game) showBankerOffer(parent fyne.Window) {
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

	// Trigger bonuses ONLY ONCE per game at a random banker offer
	// 30% chance to trigger bonus if not already offered
	if !g.bonusOffered && g.banker.r.Float64() < 0.30 {
		g.bonusOffered = true
		// Show bonuses BEFORE banker offer
		g.showBonusSequence(parent, remaining)
		return
	}

	// Normal banker offer flow (no bonus)
	if g.banker.OfferSwap() {
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

	offer := g.banker.CalculateOffer(remaining)

	// Show the offer dialog
	g.showOfferDialog(parent, offer)
}

// Show bonuses in sequence BEFORE banker offer
func (g *Game) showBonusSequence(parent fyne.Window, remaining []int) {
	hasMultiplier := g.bonus.HasMultiplier()
	hasAdditive := g.bonus.HasAdditive()

	if hasMultiplier && hasAdditive {
		// Show multiplier first, then additive, then banker offer
		g.bonus.TriggerMultiplierWithCallback(parent, func() {
			g.bonus.TriggerAdditiveWithCallback(parent, func() {
				// After both bonuses, proceed with banker offer
				g.proceedWithBankerOffer(parent, remaining)
			})
		})
	} else if hasMultiplier {
		// Show multiplier, then banker offer
		g.bonus.TriggerMultiplierWithCallback(parent, func() {
			g.proceedWithBankerOffer(parent, remaining)
		})
	} else if hasAdditive {
		// Show additive, then banker offer
		g.bonus.TriggerAdditiveWithCallback(parent, func() {
			g.proceedWithBankerOffer(parent, remaining)
		})
	} else {
		// No bonuses available, proceed directly
		g.proceedWithBankerOffer(parent, remaining)
	}
}

func (g *Game) proceedWithBankerOffer(parent fyne.Window, remaining []int) {
	// If banker proposes swap
	if g.banker.OfferSwap() {
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

	offer := g.banker.CalculateOffer(remaining)

	// Show bonus value being applied if there is one
	if g.bonus.HasPendingBonus() {
		originalOffer := offer
		offer = g.bonus.Apply(offer)
		bonusMsg := g.bonus.GetBonusDescription()

		// Only show bonus dialog if there's actually a bonus description
		if bonusMsg != "" {
			d := dialog.NewInformation("🎁 Bonus Applied!",
				fmt.Sprintf("%s\nOriginal Offer: $%d\nNew Offer: $%d", bonusMsg, originalOffer, offer),
				parent)
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

// Helper function to show offer dialog with custom buttons
func (g *Game) showOfferDialog(parent fyne.Window, offer int) {
	// Create buttons with symbols
	acceptBtn := widget.NewButton("✓ Accept", nil)
	declineBtn := widget.NewButton("✗ Decline", nil)

	// Set colors: Accept = Blue, Decline = Grey
	acceptBtn.Importance = widget.HighImportance    // Blue
	declineBtn.Importance = widget.MediumImportance // Grey

	content := widget.NewLabel(fmt.Sprintf("💰 The Banker offers you: $%d\nMeal or No Meal?", offer))
	// Accept on LEFT, Decline on RIGHT
	buttons := container.NewHBox(acceptBtn, declineBtn)
	dialogContent := container.NewVBox(content, buttons)

	dlg := dialog.NewCustomWithoutButtons("Banker's Offer", dialogContent, parent)

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
	var msg string
	if g.itemNames[g.playerTray] != "" {
		msg = fmt.Sprintf("🍽️ Your tray (Tray %d) contains: %s", g.playerTray+1, g.itemNames[g.playerTray])
	} else {
		msg = fmt.Sprintf("💰 Your tray (Tray %d) contains: $%d", g.playerTray+1, g.trayValues[g.playerTray])
	}

	d := dialog.NewInformation("Final Reveal", msg, parent)
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
	var msg string
	if g.itemNames[g.playerTray] != "" {
		msg = fmt.Sprintf("💵 You accepted the deal: $%d\n🍽️ Your tray (Tray %d) contained: %s", offer, g.playerTray+1, g.itemNames[g.playerTray])
	} else {
		msg = fmt.Sprintf("💵 You accepted the deal: $%d\n💰 Your tray (Tray %d) contained: $%d", offer, g.playerTray+1, g.trayValues[g.playerTray])
	}

	d := dialog.NewInformation("Game Over - Deal Accepted!", msg, parent)
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
