// main.go
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

const NUM_CASES = 26

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
	caseValues       []int
	caseReplaced     []int    // if case had an item, stores the numeric value removed
	itemNames        []string // "" if none
	playerCase       int
	playerCaseButton *widget.Button // visual representation of player's case
	openedCasesCount int
	openedValues     map[int]bool
	banker           *Banker
	bonus            *BonusManager
	bonusOffered     bool // track if bonus has been offered this game
}

func NewGame() *Game {
	return &Game{
		playerCase:   -1,
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

	g.caseValues = make([]int, NUM_CASES)
	for i := 0; i < NUM_CASES; i++ {
		g.caseValues[i] = sh[i%len(sh)]
	}

	// init arrays
	g.itemNames = make([]string, NUM_CASES)
	g.caseReplaced = make([]int, NUM_CASES)
	for i := range g.caseReplaced {
		g.caseReplaced[i] = -1
	}

	// choose 0..3 item replacements (avoid first/last)
	numItems := r.Intn(4)
	replace := map[int]bool{}
	for len(replace) < numItems {
		idx := r.Intn(NUM_CASES)
		if idx == 0 || idx == NUM_CASES-1 {
			continue
		}
		if !replace[idx] {
			replace[idx] = true
		}
	}

	// sample pool (shorter list - matches Java's usage of random pick)
	pool := []string{
		"Luxury Watch", "Smartphone", "Laptop", "Vacation Package", "TV", "Gaming Console",
		"Bicycle", "Headphones", "Gift Card", "Camera", "Jewelry", "Car Rental",
		"Concert Tickets", "Spa Voucher", "Restaurant Meal", "Fitness Tracker", "Drone", "Tablet",
	}

	for idx := range replace {
		name := pool[r.Intn(len(pool))]
		g.itemNames[idx] = name
		g.caseReplaced[idx] = g.caseValues[idx]
		g.caseValues[idx] = -1 // mark as item
	}

	// Build display sidebar in VALUES order; mark removed numeric values as ITEM PRICE sentinel
	display := make([]int, len(VALUES))
	copy(display, VALUES)

	removed := map[int]bool{}
	for _, v := range g.caseReplaced {
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
			ltext = "ITEM PRICE"
		}
		g.leftLabels[i] = widget.NewLabel(ltext)

		rtext := fmt.Sprintf("$%d", display[i+half])
		if display[i+half] == -999999 {
			rtext = "ITEM PRICE"
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

	g.gridButtons = make([]*widget.Button, NUM_CASES)
	grid := container.NewGridWithColumns(6)
	for i := 0; i < NUM_CASES; i++ {
		index := i
		btn := widget.NewButton(fmt.Sprintf("Case %d", i+1), func() {
			g.onCaseClicked(a, index)
		})
		btn.Importance = widget.HighImportance
		g.gridButtons[i] = btn
		grid.Add(btn)
	}

	center := container.NewHBox(left, grid, right)
	return center
}

func (g *Game) onCaseClicked(a fyne.App, idx int) {
	w := g.win

	// First pick → player's case
	if g.playerCase == -1 {
		g.playerCase = idx

		// Create a visual representation of player's case (same size as other cases)
		g.playerCaseButton = widget.NewButton(fmt.Sprintf("Case %d", idx+1), nil)
		g.playerCaseButton.Importance = widget.HighImportance

		// IMPORTANT: Disable the button BEFORE rebuilding UI
		g.gridButtons[idx].Disable()

		// bottom indicator with the case button
		bottom := container.NewCenter(
			container.NewHBox(
				widget.NewLabel("My Case: "),
				g.playerCaseButton,
			),
		)
		content := g.setupUI(a)
		// Make sure the disabled state persists
		g.gridButtons[idx].Disable()

		w.SetContent(container.NewBorder(
			container.NewCenter(widget.NewLabel("Deal or No Deal")),
			bottom,
			nil,
			nil,
			container.NewCenter(content),
		))

		dialog.ShowInformation("Your Case",
			fmt.Sprintf("You chose Case %d. This is your case until the end!", idx+1), w)
		return
	}

	// prevent re-opening player's case
	if idx == g.playerCase {
		dialog.ShowInformation("Not Allowed", "That's your case! You can't open it yet.", w)
		return
	}

	// open chosen case
	g.gridButtons[idx].Disable()
	g.openedCasesCount++

	var contentText string
	if g.itemNames[idx] != "" {
		contentText = fmt.Sprintf("You opened Case %d\nIt contained: Item — %s", idx+1, g.itemNames[idx])
	} else {
		contentText = fmt.Sprintf("You opened Case %d\nIt contained: $%d", idx+1, g.caseValues[idx])
	}

	// Show case opened dialog, then proceed with banker offer in callback
	d := dialog.NewInformation("Case Opened", contentText, w)
	d.SetOnClosed(func() {
		// mark sidebar
		g.markPriceAsOpened(idx)

		// Check if it's time for banker offer (every 3 cases)
		if g.openedCasesCount%3 == 0 {
			g.showBankerOffer(w)
		} else if g.getUnopenedCount() == 1 {
			// final reveal when 1 left (not including player's case)
			g.showFinalReveal(w)
		}
	})
	d.Show()
}

func (g *Game) markPriceAsOpened(caseIndex int) {
	// If this case originally had an item, mark the removed numeric value slot
	if g.caseReplaced[caseIndex] != -1 {
		removed := g.caseReplaced[caseIndex]
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
	val := g.caseValues[caseIndex]
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
	for i := 0; i < NUM_CASES; i++ {
		if i == g.playerCase {
			continue // Don't count player's case
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
	// Include player's case in remaining values
	for i := 0; i < NUM_CASES; i++ {
		// skip opened (disabled) cases, but include player's case
		if i != g.playerCase && g.gridButtons[i].Disabled() {
			continue
		}
		// only include numeric values (skip items)
		if g.caseValues[i] != -1 {
			remaining = append(remaining, g.caseValues[i])
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

		content := widget.NewLabel("The Banker offers to swap your case with another unopened one. Swap?")
		// Accept on LEFT, Decline on RIGHT
		buttons := container.NewHBox(acceptBtn, declineBtn)
		dialogContent := container.NewVBox(content, buttons)

		dlg := dialog.NewCustomWithoutButtons("Banker's Offer", dialogContent, parent)

		// Accept button = do the swap
		acceptBtn.OnTapped = func() {
			dlg.Hide()
			g.swapCase(parent)
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

func (g *Game) swapCase(parent fyne.Window) {
	// build list of available unopened cases
	options := []string{}
	for i := 0; i < NUM_CASES; i++ {
		if i != g.playerCase && !g.gridButtons[i].Disabled() {
			options = append(options, fmt.Sprintf("%d", i+1))
		}
	}
	if len(options) == 0 {
		dialog.ShowInformation("Swap", "No unopened cases available to swap.", parent)
		return
	}

	selectW := widget.NewSelect(options, func(s string) {})
	selectW.PlaceHolder = "Choose case number"

	swapBtn := widget.NewButton("Swap", nil)
	swapBtn.Importance = widget.HighImportance // Blue button

	dialogContent := container.NewVBox(
		widget.NewLabel("Choose a case to swap with:"),
		selectW,
		container.NewCenter(swapBtn), // Center the button
	)

	dlg := dialog.NewCustomWithoutButtons("Swap Case", dialogContent, parent)

	swapBtn.OnTapped = func() {
		if selectW.Selected != "" {
			chosen, _ := strconv.Atoi(selectW.Selected)
			newIdx := chosen - 1
			oldPlayerCase := g.playerCase

			// swap caseValues, itemNames, caseReplaced
			g.caseValues[oldPlayerCase], g.caseValues[newIdx] = g.caseValues[newIdx], g.caseValues[oldPlayerCase]
			g.itemNames[oldPlayerCase], g.itemNames[newIdx] = g.itemNames[newIdx], g.itemNames[oldPlayerCase]
			g.caseReplaced[oldPlayerCase], g.caseReplaced[newIdx] = g.caseReplaced[newIdx], g.caseReplaced[oldPlayerCase]

			// Update player case to new index
			g.playerCase = newIdx

			// Enable old case in grid, disable new case in grid
			g.gridButtons[oldPlayerCase].Enable()
			g.gridButtons[newIdx].Disable()

			// Update player case button display on the right
			g.playerCaseButton.SetText(fmt.Sprintf("Case %d", newIdx+1))

			g.refreshLabels()

			dlg.Hide()
			dialog.ShowInformation("Swap Completed",
				fmt.Sprintf("You swapped to Case %d", g.playerCase+1), parent)
		}
	}

	dlg.Show()
}
func (g *Game) refreshLabels() {
	half := len(VALUES) / 2

	// 🚨 FIX: Rebuild the display array to properly handle "ITEM PRICE" markers
	display := make([]int, len(VALUES))
	copy(display, VALUES)

	// Mark replaced values with sentinel
	removed := map[int]bool{}
	for _, v := range g.caseReplaced {
		if v != -1 {
			removed[v] = true
		}
	}
	for i := 0; i < len(display); i++ {
		if removed[display[i]] {
			display[i] = -999999 // sentinel for ITEM PRICE
		}
	}

	for i := 0; i < half; i++ {
		valLeft := display[i]
		var textLeft string
		if valLeft == -999999 {
			textLeft = "ITEM PRICE"
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
			textRight = "ITEM PRICE"
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
	if g.itemNames[g.playerCase] != "" {
		msg = fmt.Sprintf("Your case (Case %d) contains: Item — %s", g.playerCase+1, g.itemNames[g.playerCase])
	} else {
		msg = fmt.Sprintf("Your case (Case %d) contains: $%d", g.playerCase+1, g.caseValues[g.playerCase])
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
	playAgainBtn := widget.NewButton("Play Again", func() {
		// Start a fresh game
		ng := NewGame()
		ng.win = parent
		ng.initialize()
		content := ng.setupUI(fyne.CurrentApp())
		parent.SetContent(container.NewBorder(
			container.NewCenter(widget.NewLabel("Deal or No Deal")),
			nil,
			nil,
			nil,
			container.NewCenter(content),
		))
	})

	closeBtn := widget.NewButton("Close", func() {
		// Close the window and quit the app
		parent.Close()
		fyne.CurrentApp().Quit()
	})

	// Create buttons container
	buttonsContainer := container.NewHBox(playAgainBtn, closeBtn)

	// Create and show dialog, store reference so we can hide it
	dlg := dialog.NewCustomWithoutButtons("Game Over", buttonsContainer, parent)

	// Update play again button to hide dialog first
	playAgainBtn.OnTapped = func() {
		dlg.Hide()
		// Start a fresh game
		ng := NewGame()
		ng.win = parent
		ng.initialize()
		content := ng.setupUI(fyne.CurrentApp())
		parent.SetContent(container.NewBorder(
			container.NewCenter(widget.NewLabel("Deal or No Deal")),
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
	if g.itemNames[g.playerCase] != "" {
		msg = fmt.Sprintf("You accepted the deal: $%d\nYour case (Case %d) contains: Item — %s", offer, g.playerCase+1, g.itemNames[g.playerCase])
	} else {
		msg = fmt.Sprintf("You accepted the deal: $%d\nYour case (Case %d) contains: $%d", offer, g.playerCase+1, g.caseValues[g.playerCase])
	}

	d := dialog.NewInformation("Game Over", msg, parent)
	d.SetOnClosed(func() {
		for _, b := range g.gridButtons {
			b.Disable()
		}
		g.showPlayAgain(parent)
	})
	d.Show()
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

		content := widget.NewLabel("The Banker offers to swap your case with another unopened one. Swap?")
		// Accept on LEFT, Decline on RIGHT
		buttons := container.NewHBox(acceptBtn, declineBtn)
		dialogContent := container.NewVBox(content, buttons)

		dlg := dialog.NewCustomWithoutButtons("Banker's Offer", dialogContent, parent)

		// Accept button = do the swap
		acceptBtn.OnTapped = func() {
			dlg.Hide()
			g.swapCase(parent)
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
			d := dialog.NewInformation("Bonus Applied!",
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

	content := widget.NewLabel(fmt.Sprintf("The Banker offers you: $%d\nDeal or No Deal?", offer))
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

// New method to show bonuses in sequence
func (g *Game) showBonusSequence(parent fyne.Window, remaining []int) {
	hasMultiplier := g.bonus.HasMultiplier()
	hasAdditive := g.bonus.HasAdditive()

	if hasMultiplier && hasAdditive {
		// Show multiplier first, then additive, then banker offer
		g.bonus.TriggerMultiplierWithCallback(parent, func() {
			g.bonus.TriggerAdditiveWithCallback(parent, func() {
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
		// No bonuses, proceed directly
		g.proceedWithBankerOffer(parent, remaining)
	}
}

func main() {
	a := app.New()
	w := a.NewWindow("Deal or No Deal - Go/Fyne")
	g := NewGame()
	g.win = w
	g.initialize()

	content := g.setupUI(a)
	w.SetContent(container.NewBorder(
		container.NewCenter(widget.NewLabel("Deal or No Deal")),
		nil,
		nil,
		nil,
		container.NewCenter(content),
	))
	w.Resize(fyne.NewSize(1000, 600))
	w.ShowAndRun()
}
