package main

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type BonusManager struct {
	random           *rand.Rand
	multiplierActive bool
	additiveActive   bool
	multiplierUsed   bool
	additiveUsed     bool
	multiplier       float64
	additive         int
}

func NewBonusManager() *BonusManager {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return &BonusManager{
		random:           r,
		multiplierActive: r.Intn(2) == 0, // 50%
		additiveActive:   r.Intn(2) == 0, // 50%
		multiplier:       1.0,
		additive:         0,
	}
}

func (bm *BonusManager) HasMultiplier() bool { return bm.multiplierActive && !bm.multiplierUsed }
func (bm *BonusManager) HasAdditive() bool   { return bm.additiveActive && !bm.additiveUsed }

// Check if there's a pending bonus to be applied
func (bm *BonusManager) HasPendingBonus() bool {
	return bm.multiplier != 1.0 || bm.additive != 0
}

// Get description of current bonus
func (bm *BonusManager) GetBonusDescription() string {
	parts := []string{}

	if bm.multiplier != 1.0 {
		parts = append(parts, fmt.Sprintf("Multiplier: %.2fx", bm.multiplier))
	}

	if bm.additive != 0 {
		parts = append(parts, fmt.Sprintf("Additive: %+d", bm.additive))
	}

	if len(parts) == 0 {
		return ""
	}

	result := ""
	for i, part := range parts {
		if i > 0 {
			result += " and "
		}
		result += part
	}
	return result
}

func (bm *BonusManager) TriggerMultiplierWithCallback(parent fyne.Window, onComplete func()) {
	if !bm.HasMultiplier() {
		if onComplete != nil {
			onComplete()
		}
		return
	}

	options := []string{}
	for i := 0; i < 5; i++ {
		q := bm.random.Intn(4) + 2 // 2–5
		if bm.random.Intn(2) == 0 {
			options = append(options, "*"+strconv.Itoa(q))
		} else {
			options = append(options, "/"+strconv.Itoa(q))
		}
	}

	bm.showBonusChoiceDialog(parent, "Multiplier Bonus", options, func(choice string) {
		if choice[0] == '*' {
			q, _ := strconv.Atoi(choice[1:])
			bm.multiplier = float64(q)
		} else {
			q, _ := strconv.Atoi(choice[1:])
			bm.multiplier = 1.0 / float64(q)
		}
		bm.multiplierUsed = true

		// Show result, then call onComplete
		d := dialog.NewInformation("Multiplier Bonus Selected", fmt.Sprintf("You got: %s", choice), parent)
		d.SetOnClosed(func() {
			if onComplete != nil {
				onComplete()
			}
		})
		d.Show()
	})
}

func (bm *BonusManager) TriggerAdditiveWithCallback(parent fyne.Window, onComplete func()) {
	if !bm.HasAdditive() {
		if onComplete != nil {
			onComplete()
		}
		return
	}

	options := []string{}
	for i := 0; i < 10; i++ {
		val := (bm.random.Intn(20) + 1) * 100 // 100–2000
		if bm.random.Intn(2) == 0 {
			options = append(options, "+"+strconv.Itoa(val))
		} else {
			options = append(options, "-"+strconv.Itoa(val))
		}
	}

	bm.showBonusChoiceDialog(parent, "Additive Bonus", options, func(choice string) {
		if choice[0] == '+' {
			v, _ := strconv.Atoi(choice[1:])
			bm.additive = v
		} else {
			v, _ := strconv.Atoi(choice[1:])
			bm.additive = -v
		}
		bm.additiveUsed = true

		// Show result, then call onComplete
		d := dialog.NewInformation("Additive Bonus Selected", fmt.Sprintf("You got: %s", choice), parent)
		d.SetOnClosed(func() {
			if onComplete != nil {
				onComplete()
			}
		})
		d.Show()
	})
}

// Keep old methods for backward compatibility
func (bm *BonusManager) TriggerMultiplier(parent fyne.Window) {
	bm.TriggerMultiplierWithCallback(parent, nil)
}

func (bm *BonusManager) TriggerAdditive(parent fyne.Window) {
	bm.TriggerAdditiveWithCallback(parent, nil)
}

func (bm *BonusManager) Apply(offer int) int {
	modified := float64(offer)

	if bm.multiplier != 1.0 {
		modified *= bm.multiplier
		bm.multiplier = 1.0
	}

	if bm.additive != 0 {
		modified += float64(bm.additive)
		bm.additive = 0
	}

	if modified < 1 {
		return 1
	}
	return int(modified)
}

func (bm *BonusManager) showBonusChoiceDialog(parent fyne.Window, title string, options []string, onChosen func(choice string)) {
	// build a grid of buttons (cases)
	grid := container.NewGridWithColumns(5)
	var chosen string

	// declare dlg here so button closures can call dlg.Hide()
	var dlg dialog.Dialog

	for i, opt := range options {
		optCopy := opt // capture loop variable
		btn := widget.NewButton(fmt.Sprintf("Case %d", i+1), func() {
			// select this option
			chosen = optCopy
			if dlg != nil {
				dlg.Hide()
				// Call onChosen immediately after hiding
				if chosen != "" {
					onChosen(chosen)
				}
			}
		})
		btn.Resize(fyne.NewSize(80, 40))
		grid.Add(btn)
	}

	// create the custom dialog (removed the confirm buttons since we select by clicking cases)
	dlg = dialog.NewCustom(title, "Cancel", grid, parent)
	dlg.Show()
}

func (bm *BonusManager) showBonusResult(parent fyne.Window, title, bonus string) {
	dialog.ShowInformation(title, fmt.Sprintf("You got: %s", bonus), parent)
}
