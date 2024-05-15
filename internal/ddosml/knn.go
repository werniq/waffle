package ddosml

import (
	"context"
	"fmt"
	"io"
	"math"
	"sync"

	"github.com/cdipaolo/goml/base"
)

type KnnClassifier struct {
	knn *knn

	mu sync.Mutex
}

func NewKNNClassifier() *KnnClassifier {
	return &KnnClassifier{
		knn: &knn{},
	}
}

func (k *KnnClassifier) EnhanceClassifierWithRequest(m *Request) {
	//TODO implement me
	panic("implement me")
}

func (k *KnnClassifier) IsRequestPotentialDDOS(ctx context.Context, m *Request) bool {
	//TODO implement me
	panic("implement me")
}

func (k *KnnClassifier) Write(writer io.Writer) error {
	//TODO implement me
	panic("implement me")
}

type knn struct {
	// Distance holds the distance
	// measure for the knn algorithm,
	// which is just a function that
	// maps 2 float64 vectors to a
	// float64
	Distance base.DistanceMeasure

	// K is the number of nearest
	// neighbors to classify based
	// on in the knn prediction
	// algorithm
	K int

	// trainingSet holds all training
	// examples, while expectedResults
	// holds the associated class of the
	// corresponding example.
	trainingSet     [][]float64
	expectedResults []float64
}

// nn represents an encapsulation
// of the Nearest Neighbor data for
// each datapoint to facilitate easy
// sorting
type nn struct {
	X []float64
	Y float64

	Distance float64
}

// NewKNN returns a pointer to the k-means
// model, which clusters given inputs in an
// unsupervised manner. The algorithm only has
// one optimization method (unless learning with
// the online variant which is more of a generalization
// than the same algorithm) so you aren't allowed
// to pass one in as an option.
//
// n is an optional parameter which (if given) assigns
// the length of the input vector.
func NewKNN(k int, trainingSet [][]float64, expectedResults []float64, distanceMeasure base.DistanceMeasure) *knn {
	return &knn{
		Distance:        distanceMeasure,
		K:               k,
		trainingSet:     trainingSet,
		expectedResults: expectedResults,
	}
}

// UpdateTrainingSet takes in a new training set (variable x.)
func (k *knn) UpdateTrainingSet(trainingSet [][]float64, expectedResults []float64) error {
	if len(trainingSet) == 0 || len(expectedResults) == 0 {
		return fmt.Errorf("Error: length of given data is 0! Need data!")
	}
	if len(trainingSet) != len(expectedResults) {
		return fmt.Errorf("Datasets given do not match in length")
	}

	k.trainingSet = trainingSet
	k.expectedResults = expectedResults

	return nil
}

// Examples returns the number of training examples (m)
// that the model currently is holding
func (k *knn) Examples() int {
	return len(k.trainingSet)
}

// insertSorted takes a array v, and inserts u into
// the list in the position such that the list is
// sorted inversely. The function will not change the length
// of v, though, such that if u would appear last
// in the combined sorted list it would just be omitted.
//
// if the length of V is less than K, then u is inserted
// without deleting the last element
//
// Assumes v has been sorted. Uses binary search.
func insertSorted(u nn, v []nn, K int) []nn {
	low := 0
	high := len(v) - 1
	for low <= high {
		mid := (low + high) / 2
		if u.Distance < v[mid].Distance {
			high = mid - 1
		} else if u.Distance >= v[mid].Distance {
			low = mid + 1
		}
	}

	if low >= len(v) && len(v) >= K {
		return v
	}

	sorted := append(v[:low], append([]nn{u}, v[low:]...)...)

	if len(v) < K {
		return sorted
	}
	return sorted[:len(v)]
}

// round rounds a float64
func round(a float64) float64 {
	if a < 0 {
		return math.Ceil(a - 0.5)
	}
	return math.Floor(a + 0.5)
}

// Predict takes in a variable x (an array of floats,) and
// finds the value of the hypothesis function given the
// current parameter vector θ
//
// if normalize is given as true, then the input will
// first be normalized to unit length. Only use this if
// you trained off of normalized inputs and are feeding
// an un-normalized input
func (k *knn) Predict(x []float64, normalize ...bool) ([]float64, error) {
	if k.K > len(k.trainingSet) {
		return nil, fmt.Errorf("Given K (%v) is greater than the length of the training set", k.K)
	}
	if len(x) != len(k.trainingSet[0]) {
		return nil, fmt.Errorf("Given x (len %v) does not match dimensions of training set", len(x))
	}

	if len(normalize) != 0 && normalize[0] {
		base.NormalizePoint(x)
	}

	// initialize the neighbors as an empty
	// slice of Neighbors. insertSorted will
	// take care of capping the neighbors at
	// K.
	neighbors := []nn{}

	// calculate nearest neighbors
	for i := range k.trainingSet {
		dist := k.Distance(x, k.trainingSet[i])
		neighbors = insertSorted(nn{
			X: k.trainingSet[i],
			Y: k.expectedResults[i],

			Distance: dist,
		}, neighbors, k.K)
	}

	// take weighted vote
	sum := 0.0
	for i := range neighbors {
		sum += neighbors[i].Y
	}

	return []float64{round(sum / float64(k.K))}, nil
}