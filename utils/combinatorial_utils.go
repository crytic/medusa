package utils

// PermutationsWithRepetition will take in an array and an integer, n, where n represents how many items need to
// be selected from the array. The function returns an array of all permutations of size n
func PermutationsWithRepetition[T any](choices []T, n int) [][]T {
	numChoices := len(choices)

	// At each iteration of the for loop below, one of the indices in counter
	// increments by one. Here is what selector looks like over a few iterations
	// [0, 0, 0, 0] -> [1, 0, 0, 0] -> ... -> [2, 1, 0, 0] -> ... -> [4, 3, 1, 0] and so on until we reach back to
	// [0, 0, 0, 0] which means all permutations have been enumerated.
	counter := make([]int, n)
	permutations := make([][]T, 0)
	for {
		// The counter will determine the order of the current permutation. The i-th value of the permutation is equal to
		// the x-th index in the choices array.
		permutation := make([]T, n)
		for i, x := range counter {
			permutation[i] = choices[x]
		}

		// Add the permutation to the list of permutations
		permutations = append(permutations, permutation)

		// This for loop will determine the next value of the counter array
		for i := 0; ; {
			// Increment the i-th index
			counter[i]++
			// If we haven't updated the i-th index of counter up to numChoices - 1, we increment that index
			if counter[i] < numChoices {
				break
			}

			// Once the i-th index is equal to numChoices, we reset counter[i] back to 0 and move on to the next index
			// with i++
			counter[i] = 0
			i++

			// Once we reach the length of the counter array, we are done with enumerating all permutations since all
			// indices in the counter array have been reset back to 0
			if i == n {
				return permutations
			}
		}
	}
}
