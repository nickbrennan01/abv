package main

import (
	"errors"

	"github.com/bhutch29/abv/model"
	"github.com/bhutch29/abv/undo"
)

// Mode is an Enum of operating modes
type Mode string

const (
	serving  Mode = "serving"
	stocking      = "stocking"
)

// ModalController supports using the GUI via distinct behavioral modes
type ModalController struct {
	currentMode Mode
	backend     model.Model
	lastBarcode string
	actor       undo.Actor
}

// New creates a new fully initialized ModalController
func New() (ModalController, error) {
	m := ModalController{}

	m.currentMode = serving

	backend, err := model.New()
	if err != nil {
		return m, err
	}
	m.backend = backend

	a := undo.NewActor()
	m.actor = a

	return m, nil
}

// GetMode returns the current operating Mode
func (c *ModalController) GetMode() Mode {
	return c.currentMode
}

// SetMode changes the current operating Mode
func (c *ModalController) SetMode(m Mode) {
	c.currentMode = m
}

// LastBarcode returns the most recently cached barcode
func (c *ModalController) LastBarcode() string {
	return c.lastBarcode
}

// GetInventory returns the currently stocked inventory
func (c *ModalController) GetInventory() []model.StockedDrink {
	result, err := c.backend.GetInventory()
	if err != nil {
		logAllError("Error getting current inventory: ", err)
	}
	return result
}

// NewDrink stores a new drink to the database and increments the drink count
func (c *ModalController) NewDrink(d model.Drink) error {
	if c.currentMode != stocking {
		return errors.New("NewDrink can only be called from stocking mode")
	}

	id, barcode := c.parseIDFromBarcode(d.Barcode)
	logAllDebug("Parsed ID and Barcode:", "ID=" + id, ", Barcode=" + barcode)
	d.Barcode = barcode

	de := model.DrinkEntry{Barcode: barcode, Quantity: 1} //TODO add quantity handling
	a := undo.NewCreateAndInputAction(d, de)
	if err := c.actor.AddAction(id, a); err != nil {
		return err
	}
	logAllInfo("Drink created and added to inventory!\n  Name:  ", d.Name, "\n  Brand: ", d.Brand)
	return nil
}

// HandleBarcode inputs/outputs a drink and returns true if the barcode already exists or returns false if the barcode does not exist
func (c *ModalController) HandleBarcode(bc string) (bool, error) {
	c.lastBarcode = bc
	id, barcode := c.parseIDFromBarcode(bc)
	logAllDebug("Parsed ID and Barcode:", "ID=" + id, ", Barcode=" + barcode)
	exists, err := c.backend.BarcodeExists(barcode)
	if err != nil {
		return false, err
	}
	if exists {
		logFile.Info("Known barcode scanned: ", barcode)
		c.handleDrink(id, barcode)
		return true, nil
	}
	return false, nil
}

func (c *ModalController) parseIDFromBarcode(bc string) (string, string) {
	// If the second character is an _, treat the first character as a scanner ID and the rest of the input as a barcode
	if len(bc) == 1 {
		return "", bc
	}
	if bc[1] == []byte("_")[0] {
		return string(bc[0]), bc[2:]
	}
	return "", bc
}
func (c *ModalController) handleDrink(id string, bc string) {
	d := model.DrinkEntry{Barcode: bc, Quantity: 1} //TODO add quantity handling

	drink, err := c.backend.GetDrinkByBarcode(d.Barcode)
	if err != nil {
		logAllError("Error creating drink. Could not get Drink information from barcode: ", err)
	}

	if c.currentMode == stocking {
		c.inputDrinks(id, d, drink)
	} else if c.currentMode == serving {
		count, err := c.backend.GetCountByBarcode(d.Barcode)
		if err != nil {
			logAllError("Could not get count by barcode: ", err)
			return
		}
		if count <= 0 {
			logAllWarn("That drink was not in the inventory!\n  Name:  ", drink.Name, "\n  Brand: ", drink.Brand)
			return
		}
		c.outputDrinks(id, d, drink)
	}
}

func (c *ModalController) outputDrinks(id string, de model.DrinkEntry, d model.Drink) {
	a := undo.NewOutputDrinksAction(de)
	logAllDebug("Adding action with id = ", id)
	if err := c.actor.AddAction(id, a); err != nil {
		logAllError("Could not remove drink from inventory: ", err)
	} else {
		logAllInfo("Drink removed from inventory!\n  Name:  ", d.Name, "\n  Brand: ", d.Brand)
	}
}

func (c *ModalController) inputDrinks(id string, de model.DrinkEntry, d model.Drink) {
	a := undo.NewInputDrinksAction(de)
	logAllDebug("Adding action with id = ", id)
	if err := c.actor.AddAction(id, a); err != nil {
		logAllError("Could not add drink to inventory: ", err)
	} else {
		logAllInfo("Drink added to inventory!\n  Name:  ", d.Name, "\n  Brand: ", d.Brand)
	}
}

// ClearInputOutputRecords wipes out all stocking and serving records
func (c *ModalController) ClearInputOutputRecords() error {
	if err := c.backend.ClearInputTable(); err != nil {
		return err
	}
	if err := c.backend.ClearOutputTable(); err != nil {
		return err
	}
	return nil
}

// Undo reverts the previous action with the given id, if any
func (c *ModalController) Undo(id string) {
	acted, err := c.actor.Undo(id)
	if err != nil {
		logAllError("Could not undo last action with id = " + id, err)
	}
	if acted {
		logAllInfo("Reverted last action with id = ", id)
	}
}

// Redo reruns the previously reverted action with the given id, if any
func (c *ModalController) Redo(id string) {
	acted, err := c.actor.Redo(id)
	if err != nil {
		logAllError("Could not redo last action with id = " + id, err)
	}
	if acted {
		logAllInfo("Redid last action with id = ", id)
	}
}
