package main

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"math/big"
	"slices"
)

const (
	MinPasswordLength = 4
	MinPasswordsCount = 1
	MaxPasswordsCount = 50
)

var (
	ErrPasswordLengthTooLow = errors.New("password length too low")
	ErrTooLowPasswordsCount = errors.New("too low passwords count")
	ErrTooBigPasswordsCount = errors.New("too big passwords count")
)

var (
	upperChars   = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	lowerChars   = []rune("abcdefghijklmnopqrstuvwxyz")
	digitChars   = []rune("0123456789")
	specialChars = []rune("!@#$%^&*")
)

func main() {
	passwords, err := generatePassword(12, 5)
	if err != nil {
		log.Fatalf("error occured: %v\n", err)
	}

	fmt.Println(passwords) // Вывод сгенерированных паролей
}

func generatePassword(length int, count int) ([]string, error) {
	passwords := make([]string, 0, count) // Слайс, который будет содержать все пароли вместе
	seen := make(map[string]struct{})     // Мапа для проверки уникальности
	var exists = struct{}{}               // Токен для проверки уникальности

	err := checkArgue(length, count) // Проверка требований к
	if err != nil {
		return nil, err
	}

	for len(passwords) < count {
		newPass := generateOnePass(length) // FIXED: Функция возвращает string c итоговым паролем

		if _, found := seen[newPass]; found {
			continue // Пароль не уникальный, продолжить генерацию паролей
		}

		passwords = append(passwords, newPass) //  Пароль уникальный, добавить его в итоговый пароль
		seen[newPass] = exists                 // Поставить отметку о том, что такой пароль уже есть
	}

	return passwords, nil
}

func checkArgue(length int, count int) error {
	switch {
	case length < MinPasswordLength:
		return ErrPasswordLengthTooLow

	case count < MinPasswordsCount:
		return ErrTooLowPasswordsCount

	case count > MaxPasswordsCount:
		return ErrTooBigPasswordsCount
	}

	return nil
}

func generateOnePass(lengthPass int) string { // Генерация одного пароля
	runesSl := make([]rune, lengthPass)

	// NOTE: Гарантированное включение обязательных символов в итоговый слайс
	runesSl[0] = getRandRune(upperChars)
	runesSl[1] = getRandRune(lowerChars)
	runesSl[2] = getRandRune(digitChars)
	runesSl[3] = getRandRune(specialChars)

	// NOTE: Далее идет генерация случайной последовательности из всех допустимых символов
	poolOfChars := slices.Concat(upperChars, lowerChars, digitChars, specialChars)

	for i := 4; i < lengthPass-1; i++ {
		runesSl[i] = getRandRune(poolOfChars)
	}

	// Не стал делать полное перемешивание для слайса. можно сделать shuffle

	return string(runesSl)
}

func getRandRune(runes []rune) rune {
	length := len(runes)
	randIndx := randNumbers(int64(length))
	return runes[randIndx]
}

func randNumbers(n int64) (randN int64) {
	// Int не может вернуть ошибку когда используется rand.Reader
	a, _ := rand.Int(rand.Reader, big.NewInt(n))
	return a.Int64()
}
