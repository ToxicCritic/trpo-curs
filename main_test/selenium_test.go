package main_test

import (
	"testing"
	"time"

	"github.com/tebeka/selenium"
)

func TestLoginChrome(t *testing.T) {
	const remoteWebDriverURL = "http://localhost:62527"

	caps := selenium.Capabilities{
		"browserName": "chrome",
	}

	wd, err := selenium.NewRemote(caps, remoteWebDriverURL)
	if err != nil {
		t.Fatalf("Не удалось создать сессию WebDriver: %v", err)
	}
	defer wd.Quit()

	if err := wd.Get("http://87.228.39.227:8080"); err != nil {
		t.Fatalf("Не удалось загрузить страницу логина: %v", err)
	}

	time.Sleep(2 * time.Second)

	loginButton, err := wd.FindElement(selenium.ByXPATH, "//*[@id='navbarNav']/ul/li[1]/a")
	if err != nil {
		t.Fatalf("Не удалось найти кнопку 'Войти': %v", err)
	}
	if err := loginButton.Click(); err != nil {
		t.Fatalf("Ошибка при нажатии на кнопку 'Войти': %v", err)
	}

	time.Sleep(2 * time.Second)

	usernameElem, err := wd.FindElement(selenium.ByXPATH, "//*[@id='form']/form/div[1]/input")
	if err != nil {
		t.Fatalf("Не удалось найти поле для имени пользователя: %v", err)
	}

	passwordElem, err := wd.FindElement(selenium.ByXPATH, "//*[@id='form']/form/div[2]/input")
	if err != nil {
		t.Fatalf("Не удалось найти поле для пароля: %v", err)
	}

	time.Sleep(1 * time.Second)

	if err := usernameElem.Clear(); err != nil {
		t.Fatalf("Ошибка очистки поля имени пользователя: %v", err)
	}
	if err := usernameElem.SendKeys("perc"); err != nil {
		t.Fatalf("Ошибка ввода имени пользователя: %v", err)
	}

	time.Sleep(1 * time.Second)

	if err := passwordElem.Clear(); err != nil {
		t.Fatalf("Ошибка очистки поля пароля: %v", err)
	}
	if err := passwordElem.SendKeys("perc"); err != nil {
		t.Fatalf("Ошибка ввода пароля: %v", err)
	}

	time.Sleep(1 * time.Second)

	submitButton, err := wd.FindElement(selenium.ByXPATH, "//*[@id='form']/form/button")
	if err != nil {
		t.Fatalf("Не удалось найти кнопку отправки формы: %v", err)
	}
	if err := submitButton.Click(); err != nil {
		t.Fatalf("Ошибка нажатия кнопки отправки формы: %v", err)
	}

	time.Sleep(1 * time.Second)

	var greetingElem selenium.WebElement
	err = wd.WaitWithTimeout(func(wd selenium.WebDriver) (bool, error) {
		elem, err := wd.FindElement(selenium.ByXPATH, "/html/body/div/h2")
		if err != nil {
			return false, nil
		}
		greetingElem = elem
		return true, nil
	}, 10*time.Second)
	if err != nil {
		t.Fatalf("Элемент с приветствием не найден: %v", err)
	}

	greetingText, err := greetingElem.Text()
	if err != nil {
		t.Fatalf("Ошибка получения текста приветствия: %v", err)
	}
	expectedGreeting := "Привет, Администратор!"
	if greetingText != expectedGreeting {
		t.Fatalf("Неверное приветствие. Ожидалось %q, получено %q", expectedGreeting, greetingText)
	}
}
