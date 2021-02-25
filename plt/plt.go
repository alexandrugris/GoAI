package plt

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"text/template"
)

type Plot struct {
	Type string
	Values [][]float64
	Name string
	Count int
}

var plots []Plot = nil
var tmpl *template.Template = nil

func min(a int, b int) int {
	if a < b {return a} else {return b}
}

func compressByMean(count int, arr []float64) []float64{

	ret := make([]float64, count)

	for i := 0; i < count; i++ {

		upperLimit := min( (i + 1) * count, len(arr) - i * count)
		cnt := upperLimit - i * count

		ret[i] = arr[i*count] / float64(cnt)

		for j := i * count + 1; j < upperLimit; j ++ {
			ret[i] += arr[j] / float64(cnt)
		}
	}

	// last one is the last value - a hack for the simulated annealing problem
	ret[count - 1] = arr[len(arr) - 1]
	return ret
}

func toPythonArray(arr []float64) string {
	sb := strings.Builder{}
	sb.WriteString("[")

	for i, v := range arr{
		sb.WriteString(fmt.Sprintf("%f", v))
		if i < len(arr) {sb.WriteString(", ")}
	}

	sb.WriteString("]")
	return sb.String()
}

func init(){

	log.Println(os.Getwd())

	fn := template.FuncMap{
		"CompressByMean" : compressByMean,
		"ToPythonArray" : toPythonArray,
	}

	tmpl = template.Must(template.New("chart_template.py").Funcs(fn).ParseFiles("chart_template.py"))
}


func LinePlot(arr []float64, name string, count int){

	v := Plot {
		Type: "line",
		Values: make([][]float64, 1),
		Name: name,
		Count: count,
	}

	v.Values[0] = arr
	plots = append(plots, v)
}

func Reset(){
	// clear the plots
	plots = nil
}

func Execute(){

	var fileName string

	func(fn *string) {
		f, err := ioutil.TempFile("./plots", "plt*.py")

		if err != nil {
			fmt.Println(err)
			return
		}

		defer f.Close()
		*fn = f.Name()

		if err:= tmpl.Execute(f, plots); err != nil{
			log.Panic(err)
		}

		Reset()

	}(&fileName)

	go func(fileName string) {
		if out, err := exec.Command("python", fileName).Output(); err != nil {
			log.Println(err)
			log.Println(out)
		} else {
			log.Println(out)
		}
	}(fileName)

}
