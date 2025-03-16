package main

import (
	"bufio"
	"fmt"
	"math/big"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fatih/color"
)

// Wandelt einen Index in ein Passwort der Länge 'length' um.
func indexToPassword(idx int, charset []byte, length int) []byte {
	password := make([]byte, length)
	csLen := len(charset)
	for i := length - 1; i >= 0; i-- {
		password[i] = charset[idx%csLen]
		idx /= csLen
	}
	return password
}

// Berechnet die Gesamtzahl der Kombinationen für eine gegebene Länge.
func totalForLength(length int, csLen int) *big.Int {
	total := big.NewInt(1)
	csLenBig := big.NewInt(int64(csLen))
	for i := 0; i < length; i++ {
		total.Mul(total, csLenBig)
	}
	return total
}

// Struktur für einen Task, der einen bestimmten Indexbereich abdeckt.
type task struct {
	length int
	start  int
	end    int
}

// Worker, der Tasks aus der Channel-Pipeline verarbeitet.
func worker(tasks <-chan task, charset []byte, target []byte, foundFlag *int32, testedCounter *int64, foundPassword *[]byte, wg *sync.WaitGroup) {
	defer wg.Done()
	batchSize := 10000
	for t := range tasks {
		// Falls das Passwort bereits gefunden wurde, Task überspringen.
		if atomic.LoadInt32(foundFlag) == 1 {
			continue
		}
		for idx := t.start; idx <= t.end; idx += batchSize {
			if atomic.LoadInt32(foundFlag) == 1 {
				break
			}
			endBatch := t.end
			if idx+batchSize-1 < t.end {
				endBatch = idx + batchSize - 1
			}
			atomic.AddInt64(testedCounter, int64(endBatch-idx+1))
			for i := idx; i <= endBatch; i++ {
				password := indexToPassword(i, charset, t.length)
				if string(password) == string(target) {
					atomic.StoreInt32(foundFlag, 1)
					*foundPassword = password
					return
				}
			}
		}
	}
}

// Formatiert eine Zahl mit Punkten als Tausendertrennzeichen
func formatNumber(number int64) string {
	in := strconv.FormatInt(number, 10)
	out := strings.Builder{}
	for i, digit := range in {
		if i > 0 && (len(in)-i)%3 == 0 {
			out.WriteByte('.')
		}
		out.WriteRune(digit)
	}
	return out.String()
}

// Berechnet die geschätzte Zeit basierend auf der Anzahl der Kombinationen und der Geschwindigkeit
func calculateEstimatedTime(totalCombinations, speed *big.Int) (seconds, minutes, hours, days, years float64) {
	totalCombinationsF := new(big.Float).SetInt(totalCombinations)
	speedF := new(big.Float).SetInt(speed)
	secondsF := new(big.Float).Quo(totalCombinationsF, speedF)
	seconds, _ = secondsF.Float64()
	minutes = seconds / 60
	hours = minutes / 60
	days = hours / 24
	years = days / 365
	return
}

// Funktion zum Durchsuchen einer Passwortliste
func searchPasswordList(filename string, target []byte, foundFlag *int32, testedCounter *int64) (bool, int) {
	file, err := os.Open(filename)
	if err != nil {
		color.Red("Fehler beim Öffnen der Datei: %v", err)
		return false, -1
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	position := 0
	startTime := time.Now()
	for {
		if atomic.LoadInt32(foundFlag) == 1 {
			elapsedTime := time.Since(startTime).Seconds()
			color.Cyan("\nGeschätzte Zeit: %.2fs\n", elapsedTime)
			color.Cyan("Durchschnittliche Geschwindigkeit: %s Wörter/Sekunde\n", formatNumber(int64(float64(atomic.LoadInt64(testedCounter))/elapsedTime)))
			return true, position
		}
		password, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		password = strings.TrimSpace(password)
		position++
		atomic.AddInt64(testedCounter, 1)
		if password == string(target) {
			atomic.StoreInt32(foundFlag, 1)
			elapsedTime := time.Since(startTime).Seconds()
			color.Cyan("\nGeschätzte Zeit: %.2fs\n", elapsedTime)
			color.Cyan("Durchschnittliche Geschwindigkeit: %s Wörter/Sekunde\n", formatNumber(int64(float64(atomic.LoadInt64(testedCounter))/elapsedTime)))
			return true, position
		}
	}

	if err != nil {
		color.Red("Fehler beim Lesen der Datei: %v", err)
	}

	return false, -1
}

func main() {
	// Starte einen Testlauf mit dem Passwort "13321332", um die durchschnittliche Geschwindigkeit zu berechnen
	testPassword := "999999999"
	testStartTime := time.Now()
	var testCounter int64 = 0
	var testFoundFlag int32 = 0
	testCharset := []byte("0123456789") // Falls das Passwort nur aus Zahlen besteht
	testMaxLength := len(testPassword)

	// Führe den Test durch
	numCPU := runtime.NumCPU()
	testTasks := make(chan task, numCPU*2)
	var testWg sync.WaitGroup
	var foundPassword []byte

	for i := 0; i < numCPU; i++ {
		testWg.Add(1)
		go worker(testTasks, testCharset, []byte(testPassword), &testFoundFlag, &testCounter, &foundPassword, &testWg)
	}

	// Generiere Tasks für die Testlänge
	totalTestCombinations := totalForLength(testMaxLength, len(testCharset))
	for start := 0; start < int(totalTestCombinations.Int64()); start += 10000 {
		end := start + 9999
		if end >= int(totalTestCombinations.Int64()) {
			end = int(totalTestCombinations.Int64()) - 1
		}
		testTasks <- task{length: testMaxLength, start: start, end: end}
	}
	close(testTasks)
	testWg.Wait()

	// Berechne die durchschnittliche Geschwindigkeit
	testElapsedTime := time.Since(testStartTime).Seconds()
	averageSpeed := int64(float64(testCounter) / testElapsedTime)

	// Setze die Geschwindigkeit für die Hauptberechnung
	speed := big.NewInt(averageSpeed) // 70% der durchschnittlichen Geschwindigkeit

	reader := bufio.NewReader(os.Stdin)
	color.Yellow("Zielpasswort: ")
	target, _ := reader.ReadString('\n')
	target = strings.TrimSpace(target)

	color.Yellow("Möchten Sie eine Passwortliste verwenden? (ja/nein): ")
	useList, _ := reader.ReadString('\n')
	useList = strings.TrimSpace(strings.ToLower(useList))

	var foundFlag int32 = 0
	var testedCounter int64 = 0

	if useList == "ja" {
		color.Yellow("Dateiname der Passwortliste: ")
		listFilename, _ := reader.ReadString('\n')
		listFilename = strings.TrimSpace(listFilename)

		found, position := searchPasswordList(listFilename, []byte(target), &foundFlag, &testedCounter)
		if found {
			color.Green("Passwort in der Liste gefunden! Position: %d\n", position)
			return
		} else {
			color.Red("Passwort nicht in der Liste gefunden.\n")
		}
	}

	color.Cyan("\nZeichensatzauswahl:")
	color.Cyan("1. Nur Zahlen")
	color.Cyan("2. Nur Buchstaben")
	color.Cyan("3. Buchstaben + Zahlen")
	color.Cyan("4. Alle Zeichen")
	color.Yellow("Wahl (1-4): ")
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	var charset []byte
	switch choice {
	case "1":
		charset = []byte("0123456789")
	case "2":
		charset = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	case "3":
		charset = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	case "4":
		charset = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~")
	default:
		color.Red("Ungültige Eingabe!\n")
		return
	}

	color.Yellow("Maximale Länge: ")
	maxLengthStr, _ := reader.ReadString('\n')
	maxLengthStr = strings.TrimSpace(maxLengthStr)
	maxLength, err := strconv.Atoi(maxLengthStr)
	if err != nil || maxLength <= 0 {
		color.Red("Ungültige Länge!\n")
		return
	}

	// Berechne die Gesamtzahl der möglichen Kombinationen für die maximale Länge
	csLen := len(charset)
	totalCombinations := totalForLength(maxLength, csLen)
	color.Cyan("Gesamtzahl der möglichen Kombinationen für eine Länge von %d: %s\n", maxLength, totalCombinations.String())

	// Beispielgeschwindigkeit: 37 Millionen Passwörter pro Sekunde
	seconds, minutes, hours, days, years := calculateEstimatedTime(totalCombinations, speed)
	color.Cyan("Geschätzte Zeit für das Durchprobieren aller Kombinationen bei %d Passwörtern/Sekunde:\n", speed)
	color.Cyan("Sekunden: %.2f\n", seconds)
	color.Cyan("Minuten: %.2f\n", minutes)
	color.Cyan("Stunden: %.2f\n", hours)
	color.Cyan("Tage: %.2f\n", days)
	color.Cyan("Jahre: %.2f\n", years)

	startTime := time.Now()
	runtime.GOMAXPROCS(numCPU)

	// Task-Channel und Worker-Pool starten.
	tasks := make(chan task, numCPU*2)
	var wg sync.WaitGroup
	for i := 0; i < numCPU; i++ {
		wg.Add(1)
		go worker(tasks, charset, []byte(target), &foundFlag, &testedCounter, &foundPassword, &wg)
	}

	// Fortschrittsanzeige in einem separaten Goroutine.
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-done:
				return
			default:
				color.Cyan("\rGesamtfortschritt: %s Passwörter getestet", formatNumber(atomic.LoadInt64(&testedCounter)))
				time.Sleep(1000 * time.Millisecond)
			}
		}
	}()

	// Erzeuge Tasks für jede Passwortlänge, ohne sie auszugeben.
	for length := 1; length <= maxLength && atomic.LoadInt32(&foundFlag) == 0; length++ {
		total := totalForLength(length, csLen)
		// Bestimme die Chunk-Größe: maximal 10000 oder total/(10*numCPU), je nachdem, was größer ist.
		chunkSize := 50000
		temp := new(big.Int).Div(total, big.NewInt(int64(10*numCPU)))
		if temp.Cmp(big.NewInt(int64(chunkSize))) == 1 {
			chunkSize = int(temp.Int64())
		}
		for start := int64(0); start < total.Int64() && atomic.LoadInt32(&foundFlag) == 0; start += int64(chunkSize) {
			end := start + int64(chunkSize) - 1
			if end >= total.Int64() {
				end = total.Int64() - 1
			}
			tasks <- task{length: length, start: int(start), end: int(end)}
		}
	}
	close(tasks)
	wg.Wait()
	close(done)

	elapsedTime := time.Since(startTime).Seconds()
	fmt.Println("\n" + strings.Repeat("=", 50))
	if atomic.LoadInt32(&foundFlag) == 1 {
		color.Green("Passwort gefunden: %s\n", string(foundPassword))
		color.Cyan("Gesamtzeit: %.2fs\n", elapsedTime)
		color.Cyan("Getestete Passwörter: %s\n", formatNumber(atomic.LoadInt64(&testedCounter)))
		color.Cyan("Durchschnittliche Geschwindigkeit: %s Passwörter/Sekunde\n", formatNumber(int64(float64(atomic.LoadInt64(&testedCounter))/elapsedTime)))
	} else {
		color.Red("Passwort nicht gefunden\n")
	}
}
