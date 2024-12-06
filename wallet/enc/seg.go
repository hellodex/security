package enc

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"time"

	mathrand "math/rand"
)

type Hexkey string

func (t Hexkey) Decode() (v []byte) {
	v, err := hex.DecodeString(string(t))
	if err != nil {
		fmt.Println(err)
	}
	return v
}

type polyfunk struct {
	complexs []uint8
}

func (p *polyfunk) evaluate(x uint8) uint8 {
	if x == 0 {
		return p.complexs[0]
	}

	degree := len(p.complexs) - 1
	out := p.complexs[degree]
	for i := degree - 1; i >= 0; i-- {
		coeff := p.complexs[i]
		out = add(mult(out, x), coeff)
	}
	return out
}

func mult(a, b uint8) (out uint8) {
	var r uint8 = 0
	var i uint8 = 8

	for i > 0 {
		i--
		r = (-(b >> i & 1) & a) ^ (-(r >> 7) & 0x1B) ^ (r + r)
	}

	return r
}

func add(a, b uint8) uint8 {
	return a ^ b
}

func div(a, b uint8) uint8 {
	if b == 0 {
		panic("divide by zero")
	}

	ret := int(mult(a, inverse(b)))

	ret = subtle.ConstantTimeSelect(subtle.ConstantTimeByteEq(a, 0), 0, ret)
	return uint8(ret)
}

func inverse(a uint8) uint8 {
	b := mult(a, a)
	c := mult(a, b)
	b = mult(c, c)
	b = mult(b, b)
	c = mult(b, c)
	b = mult(b, b)
	b = mult(b, b)
	b = mult(b, c)
	b = mult(b, b)
	b = mult(a, b)

	return mult(b, b)
}

func interpolatePolynomial(x_samples, y_samples []uint8, x uint8) uint8 {
	limit := len(x_samples)
	var result, basis uint8
	for i := 0; i < limit; i++ {
		basis = 1
		for j := 0; j < limit; j++ {
			if i == j {
				continue
			}
			num := add(x, x_samples[j])
			denom := add(x_samples[i], x_samples[j])
			term := div(num, denom)
			basis = mult(basis, term)
		}
		group := mult(y_samples[i], basis)
		result = add(result, group)
	}
	return result
}

func makePolyfunk(intercept, degree uint8) (polyfunk, error) {
	p := polyfunk{
		complexs: make([]byte, degree+1),
	}

	p.complexs[0] = intercept

	if _, err := rand.Read(p.complexs[1:]); err != nil {
		return p, err
	}

	return p, nil
}

func split(secret []byte, parts, threshold int) ([][]byte, error) {
	if parts < threshold {
		return nil, fmt.Errorf("key ps should not be less than thres")
	}
	if parts > 255 {
		return nil, fmt.Errorf("key ps cannot overlimit 255")
	}
	if threshold < 2 {
		return nil, fmt.Errorf("threshold must be gt 2")
	}
	if threshold > 255 {
		return nil, fmt.Errorf("threshold cannot overlimit 255")
	}
	if len(secret) == 0 {
		return nil, fmt.Errorf("cannot split empty key ps")
	}

	mathrand.Seed(time.Now().UnixNano())
	xCoordinates := mathrand.Perm(255)

	out := make([][]byte, parts)
	for idx := range out {
		out[idx] = make([]byte, len(secret)+1)
		out[idx][len(secret)] = uint8(xCoordinates[idx]) + 1
	}

	for idx, val := range secret {
		p, err := makePolyfunk(val, uint8(threshold-1))
		if err != nil {
			return nil, fmt.Errorf("failed for generating polynomial: %w", err)
		}

		for i := 0; i < parts; i++ {
			x := uint8(xCoordinates[i]) + 1
			y := p.evaluate(x)
			out[i][idx] = y
		}
	}
	return out, nil
}

func combine(parts [][]byte) ([]byte, error) {
	if len(parts) < 2 {
		return nil, fmt.Errorf("lt two key ps cannot revert the secret")
	}

	firstPartLen := len(parts[0])
	if firstPartLen < 2 {
		return nil, fmt.Errorf("parts must be gt two bytes")
	}
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) != firstPartLen {
			return nil, fmt.Errorf("all parts must be the same length")
		}
	}

	secret := make([]byte, firstPartLen-1)

	x_samples := make([]uint8, len(parts))
	y_samples := make([]uint8, len(parts))

	checkMap := map[byte]bool{}
	for i, part := range parts {
		samp := part[firstPartLen-1]
		if exists := checkMap[samp]; exists {
			return nil, fmt.Errorf("duplicate part, not al")
		}
		checkMap[samp] = true
		x_samples[i] = samp
	}

	for idx := range secret {
		for i, part := range parts {
			y_samples[i] = part[idx]
		}

		val := interpolatePolynomial(x_samples, y_samples, 0)

		secret[idx] = val
	}
	return secret, nil
}

func recover(shares []Hexkey) (string, error) {
	var bshares [][]byte
	for _, v := range shares {
		bshares = append(bshares, v.Decode())
	}
	recoveredSecret, err := combine(bshares)
	if err != nil {
		return "", fmt.Errorf("failed to recover secret: %w", err)
	}
	return string(recoveredSecret), nil
}

func Split(secret string, totalShares, threshold int) ([]Hexkey, error) {
	secretBytes := []byte(secret)

	shares, err := split(secretBytes, totalShares, threshold)
	if err != nil {
		return nil, fmt.Errorf("failed to split secret: %w", err)
	}
	keyShares := make([]Hexkey, 0)
	for _, v := range shares {
		keyShares = append(keyShares, Hexkey(hex.EncodeToString(v)))
	}
	return keyShares, nil
}
