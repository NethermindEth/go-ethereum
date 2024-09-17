package crypto

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/NethermindEth/weierstrass"
)

// NewCurve creates new Weierstrass (short form) curve
// y^2 = x^3 + ax + b mod p
func NewCurve(a, b, p *big.Int) weierstrass.Curve {
	c := weierstrass.NewCurve(a, b, p)
	return c
}

func NewPoint(x, y *big.Int, c weierstrass.Curve) (*weierstrass.Point, error) {
	// TODO: implement this in underlying lib

	// if xIsNotPartOfField := x.Cmp(c.P) > 0; xIsNotPartOfField {
	// 	return nil, fmt.Errorf("Provided X coordinate is not within prime field range, got %d, field size is %d", x, c.P)
	// }
	//
	// if yIsNotPartOfField := y.Cmp(c.P) > 0; yIsNotPartOfField {
	// 	return nil, fmt.Errorf("Provided X coordinate is not within prime field range, got %d, field size is %d", y, c.P)
	// }

	p := weierstrass.NewPoint(x, y)

	if c.IsOnCurve(p) {
		return &p, nil
	}

	return nil, errors.New("Provided point is not on curve")
}

func MulAdd(p, q weierstrass.Point, k, u *big.Int, c weierstrass.Curve) weierstrass.Point {
	return c.ScalarMulAddPoints(p, q, k, u)
}
