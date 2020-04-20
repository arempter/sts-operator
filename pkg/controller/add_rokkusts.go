package controller

import (
	"github.com/arempter/sts-operator/pkg/controller/rokkusts"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, rokkusts.Add)
}
