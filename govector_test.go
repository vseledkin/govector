package govector

import (
	"fmt"
	"testing"
	"time"
)

func TestGet(t *testing.T) {
	//m, e := NewManifold("/Volumes/data/fasttextmodel/RUEVENTOS/ru.cbow")
	m, e := NewManifold("/Users/vseledkin/ru.cbow")
	if e != nil {
		t.Fatal(fmt.Errorf("Invalid govector directory: [%s]", e))
	}
	e = m.Open()
	if e != nil {
		t.Fatal(fmt.Errorf("Invalid govector directory: [%s]", e))
	}
	defer m.Close()
	get := func(word string) {
		start := time.Now()
		v, e := m.GetVector(word)
		if e != nil {
			t.Fatal(fmt.Errorf("ERR Cannot get vector for: [%s] - %s", word, e))
		}
		t.Logf("%s %#v %f s.", word, v, time.Now().Sub(start).Seconds())
	}
	get("можно")
	get("Путdин")

}

func TestSplit(t *testing.T) {
	s := "Путин"
	ngrams := сomputeNGrams(s, 3, 6)
	t.Logf("%#v", ngrams)
	//if split != want {
	//t.Fatalf("Split failed want\n%s\ngot\n%s\n", want, split)
	//}
}

func TestOptics(t *testing.T) {
	//m, e := NewManifold("/Volumes/data/fasttextmodel/RUEVENTOS/ru.cbow")
	m, e := NewManifold("/Users/vseledkin/ru.cbow")
	if e != nil {
		t.Fatal(fmt.Errorf("Invalid govector directory: [%s]", e))
	}
	e = m.Open()
	if e != nil {
		t.Fatal(fmt.Errorf("Invalid govector directory: [%s]", e))
	}
	defer m.Close()

	m.ComputeClusters(0.3, 1)

}
