package metadata

type Metadata struct {
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
