package types

import (
	"fmt"
	"hash/fnv"
	"slices"
)

type Alert struct {
	Annotations map[string]string `json:"annotations"`
	Labels      map[string]string `json:"labels"`
	StartsAt    string            `json:"startsAt"`
	Status      string            `json:"status"`
}

func (a Alert) LabelsFingerprint() string {
	sum := fnv.New64a()

	sortedLabels := make([]string, 0, len(a.Labels))
	for l := range a.Labels {
		sortedLabels = append(sortedLabels, l)
	}
	slices.Sort(sortedLabels)
	for _, k := range sortedLabels {
		sum.Write([]byte(k))
		sum.Write([]byte{255})
		sum.Write([]byte(a.Labels[k]))
		sum.Write([]byte{255})
	}

	sum.Write([]byte(a.StartsAt))
	sum.Write([]byte{255})

	return fmt.Sprintf("%016x", sum.Sum64())
}

func (a Alert) Fingerprint() string {
	sum := fnv.New64a()

	sortedAnnotations := make([]string, 0, len(a.Annotations))
	for l := range a.Labels {
		sortedAnnotations = append(sortedAnnotations, l)
	}
	slices.Sort(sortedAnnotations)
	for _, k := range sortedAnnotations {
		sum.Write([]byte(k))
		sum.Write([]byte{255})
		sum.Write([]byte(a.Annotations[k]))
		sum.Write([]byte{255})
	}

	sortedLabels := make([]string, 0, len(a.Labels))
	for l := range a.Labels {
		sortedLabels = append(sortedLabels, l)
	}
	slices.Sort(sortedLabels)
	for _, k := range sortedLabels {
		sum.Write([]byte(k))
		sum.Write([]byte{255})
		sum.Write([]byte(a.Labels[k]))
		sum.Write([]byte{255})
	}

	sum.Write([]byte(a.StartsAt))
	sum.Write([]byte{255})
	sum.Write([]byte(a.Status))
	sum.Write([]byte{255})

	return fmt.Sprintf("%016x", sum.Sum64())
}
