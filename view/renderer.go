package view

type Viewer interface {
	Do(yearMonth string) error
}
