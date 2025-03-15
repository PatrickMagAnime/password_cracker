package main

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Wandelt einen Index in ein Passwort der Länge 'length' um.
func indexToPassword(idx int, charset []rune, length int) string {
	password := make([]rune, length)
	csLen := len(charset)
	for i := length - 1; i >= 0; i-- {
		password[i] = charset[idx%csLen]
		idx /= csLen
	}
	return string(password)
}

// Berechnet die Gesamtzahl der Kombinationen für eine gegebene Länge.
func totalForLength(length, csLen int) int64 {
	total := int64(1)
	for i := 0; i < length; i++ {
		total *= int64(csLen)
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
func worker(tasks <-chan task, charset []rune, target string, foundFlag *int32, testedCounter *int64, wg *sync.WaitGroup) {
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
				if indexToPassword(i, charset, t.length) == target {
					atomic.StoreInt32(foundFlag, 1)
					return
				}
			}
		}
	}
}

// Formatiert eine Zahl mit Punkten als Tausendertrennzeichen
func formatNumber(number int64) string {
	in := strconv.FormatInt(number, 10)
	out := ""
	for i, digit := range in {
		if i > 0 && (len(in)-i)%3 == 0 {
			out += "."
		}
		out += string(digit)
	}
	return out
}

// Berechnet die geschätzte Zeit basierend auf der Anzahl der Kombinationen und der Geschwindigkeit
func calculateEstimatedTime(totalCombinations, speed int64) (seconds, hours, days, years float64) {
	seconds = float64(totalCombinations) / float64(speed)
	hours = seconds / 3600
	days = hours / 24
	years = days / 365
	return
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Zielpasswort: ")
	target, _ := reader.ReadString('\n')
	target = strings.TrimSpace(target)

	fmt.Println("\nZeichensatzauswahl:")
	fmt.Println("1. Nur Zahlen")
	fmt.Println("2. Nur Buchstaben")
	fmt.Println("3. Buchstaben + Zahlen")
	fmt.Println("4. Alle Zeichen")
	fmt.Print("Wahl (1-4): ")
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	var charset []rune
	switch choice {
	case "1":
		charset = []rune("0123456789")
	case "2":
		charset = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	case "3":
		charset = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	case "4":
		charset = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~")
	default:
		fmt.Println("Ungültige Eingabe!")
		return
	}

	fmt.Print("Maximale Länge: ")
	maxLengthStr, _ := reader.ReadString('\n')
	maxLengthStr = strings.TrimSpace(maxLengthStr)
	maxLength, err := strconv.Atoi(maxLengthStr)
	if err != nil || maxLength <= 0 {
		fmt.Println("Ungültige Länge!")
		return
	}

	// Berechne die Gesamtzahl der möglichen Kombinationen für die maximale Länge
	csLen := len(charset)
	totalCombinations := totalForLength(maxLength, csLen)
	fmt.Printf("Gesamtzahl der möglichen Kombinationen für eine Länge von %d: %s\n", maxLength, formatNumber(totalCombinations))

	// Beispielgeschwindigkeit: 37 Millionen Passwörter pro Sekunde
	speed := int64(28000000)
	seconds, hours, days, years := calculateEstimatedTime(totalCombinations, speed)
	fmt.Printf("Geschätzte Zeit für das Durchprobieren aller Kombinationen bei %d Passwörtern/Sekunde:\n", speed)
	fmt.Printf("Sekunden: %.2f\n", seconds)
	fmt.Printf("Stunden: %.2f\n", hours)
	fmt.Printf("Tage: %.2f\n", days)
	fmt.Printf("Jahre: %.2f\n", years)

	startTime := time.Now()
	var foundFlag int32 = 0
	var testedCounter int64 = 0

	numCPU := runtime.NumCPU()
	runtime.GOMAXPROCS(numCPU)

	// Task-Channel und Worker-Pool starten.
	tasks := make(chan task, numCPU*2)
	var wg sync.WaitGroup
	for i := 0; i < numCPU; i++ {
		wg.Add(1)
		go worker(tasks, charset, target, &foundFlag, &testedCounter, &wg)
	}

	// Fortschrittsanzeige in einem separaten Goroutine.
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-done:
				return
			default:
				fmt.Printf("\rGesamtfortschritt: %s Passwörter getestet", formatNumber(atomic.LoadInt64(&testedCounter)))
				time.Sleep(500 * time.Millisecond)
			}
		}
	}()

	// Erzeuge Tasks für jede Passwortlänge.
	for length := 1; length <= maxLength && atomic.LoadInt32(&foundFlag) == 0; length++ {
		total := totalForLength(length, csLen)
		// Bestimme die Chunk-Größe: maximal 10000 oder total/(10*numCPU), je nachdem, was größer ist.
		chunkSize := 10000
		temp := int(total) / (10 * numCPU)
		if temp > chunkSize {
			chunkSize = temp
		}
		for start := 0; start < int(total) && atomic.LoadInt32(&foundFlag) == 0; start += chunkSize {
			end := start + chunkSize - 1
			if end >= int(total) {
				end = int(total) - 1
			}
			tasks <- task{length: length, start: start, end: end}
		}
	}
	close(tasks)
	wg.Wait()
	close(done)

	elapsedTime := time.Since(startTime).Seconds()
	fmt.Println("\n" + strings.Repeat("=", 50))
	if atomic.LoadInt32(&foundFlag) == 1 {
		fmt.Printf("Passwort gefunden!\n")
		fmt.Printf("Gesamtzeit: %.2fs\n", elapsedTime)
		fmt.Printf("Getestete Passwörter: %s\n", formatNumber(atomic.LoadInt64(&testedCounter)))
		fmt.Printf("Durchschnittliche Geschwindigkeit: %s Passwörter/Sekunde\n", formatNumber(int64(float64(atomic.LoadInt64(&testedCounter))/elapsedTime)))
	} else {
		fmt.Println("Passwort nicht gefunden")
	}
}
