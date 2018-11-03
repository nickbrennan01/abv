package main

import (
	"github.com/bhutch29/abv/model"
	"fmt"
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

// HandleBarcode inputs/outputs a drink and returns true if the barcode already exists or returns false if the barcode does not exist
func (c *ModalController) HandleBarcode(bc string) (bool, error) {
	c.lastBarcode = bc
	exists, err := c.backend.BarcodeExists(bc)
	if err != nil {
		return false, err
	}
	if exists {
		logFile.Info("Known barcode scanned: ", bc)
		c.handleDrink(bc)
		return true, nil
	}
	return false, nil
}

// NewDrink stores a new drink to the database and increments the drink count
func (c *ModalController) NewDrink(d model.Drink) error {
	if c.currentMode != stocking {
		return fmt.Errorf("NewDrink can only be called from stocking mode")
	}
	if _, err := c.backend.CreateDrink(d); err != nil {
		return err
	}
	c.handleDrink(d.Barcode)
	return nil
}

func (c *ModalController) handleDrink(bc string) {
	d := model.DrinkEntry{Barcode: bc, Quantity: 1} //TODO add quantity handling

	drink, err := c.backend.GetDrinkByBarcode(d.Barcode)
	if err != nil {
		logGui.Error(err)
		logFile.Error(err)
	}

	if c.currentMode == stocking {
		logGui.Info("Drink added to inventory! Name: ", drink.Name, ", Brand: ", drink.Brand)
		logFile.Info("Drink added to inventory! Name: ", drink.Name, ", Brand: ", drink.Brand)
		c.backend.InputDrinks(d)
	} else if c.currentMode == serving {
		count, err := c.backend.GetCountByBarcode(d.Barcode)
		if err != nil {
			logGui.Error("Could not get count by barcode: ", err)
			logFile.Error("Could not get count by barcode: ", err)
			return
		}
		if count <= 0 {
			logGui.Warn("That drink was not in the inventory! Name: ", drink.Name, ", Brand: ", drink.Brand)
			logFile.Warn("Drink scanned out that was not in the inventory!", drink)
			return
		}
		logGui.Info("Drink removed from inventory! Name: ", drink.Name, ", Brand: ", drink.Brand)
		logFile.Info("Drink removed from inventory! Name: ", drink.Name, ", Brand: ", drink.Brand)
		c.backend.OutputDrinks(d)
	}
}
