package main

import (
	"GoAI/plt"
	"fmt"
	"github.com/tfriedel6/canvas/sdlcanvas"
	"math"
	"math/rand"
)

type Point struct {
	X float64
	Y float64
}

type Connection struct {
	Start int
	End   int
}

func (p *Point) Subtract(other *Point) Point {
	return Point{
		X: p.X - other.X,
		Y: p.Y - other.Y,
	}
}

func (p *Point) DistanceTo(other *Point) float64 {
	d := other.Subtract(p)
	return math.Sqrt(d.X*d.X + d.Y*d.Y)
}

type ConnsCollection struct {
	Points []Point
	Conns  []Connection

	// map ending to index in Conn
	endsIn []int
}

func (cc *ConnsCollection) BuildEndsInMap() {

	if cc.endsIn == nil || len(cc.endsIn) != len(cc.Conns) {
		cc.endsIn = make([]int, len(cc.Conns))
	}

	for i, cn := range cc.Conns {
		cc.endsIn[cn.End] = i
	}
}

func (cc *ConnsCollection) ComputeDistance() (float64, bool) {
	d := 0.0
	for _, c := range cc.Conns {
		if c.Start >= len(cc.Points) || c.End >= len(cc.Points) {
			return -1, false
		}
		d += cc.Points[c.Start].DistanceTo(&cc.Points[c.End])
	}
	return d, true
}

func (cc *ConnsCollection) DuplicateConnectionsTo(other **ConnsCollection) {

	if *other == nil {
		*other = &ConnsCollection{
			Points: cc.Points,
			Conns:  make([]Connection, len(cc.Conns)),
			endsIn: make([]int, len(cc.endsIn)),
		}
	}

	copy((*other).Conns, cc.Conns)
	copy((*other).endsIn, cc.endsIn)

}

func (cc *ConnsCollection) ComputeNewPath() float64 {

	conns := cc.Conns

	if len(conns) <= 1 {
		return 0.0
	}

	i1 := rand.Int() % len(conns)
	i2 := rand.Int() % len(conns)
	if i1 == i2 {
		i2++
		if i2 == len(conns) {
			i2 = 0
		}
	}

	p1 := &conns[i1]
	p2 := &conns[i2]

	// swap edges
	p1.End, p2.Start = p2.Start, p1.End

	for idx := p1.End; idx != p2.Start; {
		c := &conns[cc.endsIn[idx]]
		c.Start, c.End = c.End, c.Start
		idx = c.End
	}

	d, _ := cc.ComputeDistance()
	return d
}

func InitConnectionsFromPoints(points []Point) *ConnsCollection {

	c := ConnsCollection{
		Points: points,
		Conns:  make([]Connection, 0, 20),
	}

	// crate a path where each point is travelled only once
	for i := range c.Points {

		s := i
		e := i + 1

		if e == len(c.Points) {
			e = 0
		}

		c.Conns = append(c.Conns, Connection{
			Start: s,
			End:   e,
		})
	}

	c.BuildEndsInMap()

	return &c
}

func TravellingSalesman(in <-chan []Point, out chan<- *ConnsCollection) {

	for {

		// read all points and only start the computation once I finished points
		points := <-in
		for len(in) > 0 {
			points = <-in
		}

		var conns, conns2, resetPoint *ConnsCollection
		conns = InitConnectionsFromPoints(points)
		conns.DuplicateConnectionsTo(&conns2)
		conns.DuplicateConnectionsTo(&resetPoint)

		d, _ := conns.ComputeDistance()
		dReset := d
		MaxDriftFromGlobalMinimum := 10 * len(points)
		countSinceReset := MaxDriftFromGlobalMinimum

		MaxIterations := 100000
		distanceEvolution := make([]float64, MaxIterations)

		for i := 0; i < MaxIterations; i++ {

			temperature := 0.1 * float64(MaxIterations-i) / float64(MaxIterations)
			temperature = math.Pow(temperature, 5)

			// switch two nodes
			d2 := conns2.ComputeNewPath()

			// found a better move
			// or the temperature is high enough to accept other moves
			if d2 < d || (d2-d)*temperature > rand.Float64() {

				if d2 > d && i > (MaxIterations/100)*50 {
					fmt.Printf("Accepted bad move: iter: %v, temp: %v, distance: %v\n", i, temperature, d2-d)
				}

				conns2.BuildEndsInMap()
				conns2.DuplicateConnectionsTo(&conns)
				d = d2

				if d < dReset {
					dReset = d
					countSinceReset = MaxDriftFromGlobalMinimum
					conns2.DuplicateConnectionsTo(&resetPoint)
				}

			} else if countSinceReset < 0 {
				d = dReset
				countSinceReset = MaxDriftFromGlobalMinimum
				resetPoint.DuplicateConnectionsTo(&conns)
				resetPoint.DuplicateConnectionsTo(&conns2)
				//fmt.Println("Reset")
			} else {
				conns.DuplicateConnectionsTo(&conns2) // re-init conns2
			}

			countSinceReset--

			// save for analysis
			distanceEvolution[i] = d
		}

		plt.Reset()
		plt.LinePlot(distanceEvolution, "Distance Evolution", 1000)

		if d > dReset {
			out <- resetPoint
		} else {
			out <- conns
		}
	}
}

func main() {
	wnd, cv, err := sdlcanvas.CreateWindow(1280, 720, "Travelling Salesman")
	if err != nil {
		panic(err)
	}
	defer wnd.Destroy()

	points := make([]Point, 0, 10)
	connections := make([]Connection, 0, 10)
	distance := 0.0

	submitPoints := make(chan []Point, 100)
	receiveConnections := make(chan *ConnsCollection)

	go TravellingSalesman(submitPoints, receiveConnections)

	wnd.MouseDown = func(btn int, x int, y int) {
		// on mouse down add new points
		points = append(points, Point{
			X: float64(x),
			Y: float64(y),
		})

		// send the points to be computed
		submitPoints <- points
	}

	wnd.KeyDown = func(scancode int, rn rune, name string) {
		switch name {
		case "Escape":
			points = make([]Point, 0, 10)
			connections = make([]Connection, 0, 10)
			distance = 0.0
		case "KeyP":
			plt.Execute() // show plot only when key is pressed
		}
	}

	wnd.MainLoop(func() {

		select {

		case cc := <-receiveConnections:
			if dd, ok := cc.ComputeDistance(); ok {
				distance = dd
				connections = cc.Conns
				fmt.Printf("New paths with distance %f\n", distance)
			}

		default:

		}

		// background
		w, h := float64(cv.Width()), float64(cv.Height())
		cv.SetFillStyle("#000")
		cv.FillRect(0, 0, w, h)

		// circles
		cv.SetStrokeStyle("#FFF")
		cv.SetLineWidth(2)
		cv.SetFillStyle(255, 0, 0)

		for _, c := range connections {
			cv.BeginPath()
			cv.MoveTo(points[c.Start].X, points[c.Start].Y)
			cv.LineTo(points[c.End].X, points[c.End].Y)
			cv.Stroke()
		}

		for _, p := range points {
			cv.BeginPath()
			cv.Arc(p.X, p.Y, 10, 0, math.Pi*2, false)
			cv.ClosePath()
			cv.Fill()
			cv.Stroke()
		}

		cv.SetFont("Righteous-Regular.ttf", 12)
		cv.FillText(fmt.Sprintf("Total distance: %f", distance), 20, 20)

	})
}
