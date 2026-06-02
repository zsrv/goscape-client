package metadata

type Metadata struct {
	// Data is the raw per-id model blob; NewModel1 reads vertex/face sections
	// out of it via offsets stored below. Java: Metadata.data (rev-244).
	Data                   []byte
	VertexCount            int
	FaceCount              int
	TexturedFaceCount      int
	VertexFlagsOffset      int
	VertexXOffset          int
	VertexYOffset          int
	VertexZOffset          int
	VertexLabelsOffset     int
	FaceVerticesOffset     int
	FaceOrientationsOffset int
	FaceColoursOffset      int
	FaceInfosOffset        int
	FacePrioritiesOffset   int
	FaceAlphasOffset       int
	FaceLabelsOffset       int
	FaceTextureAxisOffset  int
}

func NewMetadata() *Metadata {
	return new(Metadata)
}
