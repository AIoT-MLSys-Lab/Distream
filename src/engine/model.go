package engine

var Model_Duration_Fraction = []float64{0.4, 0.6, 0.8, 1.0}

type Model struct {
	fractions   []float64
	SelectModel int
}

func NewModel() *Model {
	return &Model{Model_Duration_Fraction, len(Model_Duration_Fraction) - 1}
}

func (m *Model) upshift() {
	m.SelectModel += 1
	if m.SelectModel >= len(m.fractions) {
		m.SelectModel = len(m.fractions) - 1
	}
	if m.SelectModel < 0 {
		m.SelectModel = 0
	}
}

func (m *Model) downshift() {
	m.SelectModel -= 1
	if m.SelectModel >= len(m.fractions) {
		m.SelectModel = len(m.fractions) - 1
	}
	if m.SelectModel < 0 {
		m.SelectModel = 0
	}
}

func (m *Model) fraction() float64 {
	return m.fractions[m.SelectModel]
}
