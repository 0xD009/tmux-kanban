package main

type panelBounds struct {
	panel focusedPanel
	x     int
	y     int
	w     int
	h     int
}

func (b panelBounds) contains(x int, y int) bool {
	return x >= b.x && x < b.x+b.w && y >= b.y && y < b.y+b.h
}

func (m model) panelAt(x int, y int) focusedPanel {
	for _, bounds := range m.panelBounds() {
		if bounds.contains(x, y) {
			return bounds.panel
		}
	}
	return panelNone
}

func (m model) panelBounds() []panelBounds {
	contentWidth := maxInt(60, m.width-4)
	contentTop := m.headerHeight()
	contentHeight := maxInt(18, m.height-contentTop)
	if m.viewMode == viewReview {
		return reviewPanelBounds(contentWidth, contentTop, contentHeight)
	}
	return treePanelBounds(contentWidth, contentTop, contentHeight)
}

func treePanelBounds(width int, top int, height int) []panelBounds {
	if width >= 140 {
		kanbanWidth := threeColumnSideWidth(width)
		activityWidth := threeColumnActivityWidth(width, kanbanWidth)
		workspaceWidth := maxInt(60, width-kanbanWidth-activityWidth-4)
		bounds := []panelBounds{{panel: panelKanban, x: 0, y: top, w: kanbanWidth, h: height}}
		bounds = append(bounds, workspacePanelBounds(kanbanWidth+2, top, workspaceWidth, height)...)
		bounds = append(bounds, panelBounds{panel: panelActivity, x: kanbanWidth + 2 + workspaceWidth + 2, y: top, w: activityWidth, h: height})
		return bounds
	}

	if width >= 104 {
		kanbanWidth := twoColumnSideWidth(width)
		workspaceWidth := maxInt(60, width-kanbanWidth-2)
		bounds := []panelBounds{{panel: panelKanban, x: 0, y: top, w: kanbanWidth, h: height}}
		bounds = append(bounds, workspacePanelBounds(kanbanWidth+2, top, workspaceWidth, height)...)
		return bounds
	}

	kanbanHeight := 0
	totalPanelHeight := height - 2
	if height >= 34 {
		totalPanelHeight = height - 4
		kanbanHeight = maxInt(8, totalPanelHeight/5)
	}
	hostHeight, previewHeight := splitWorkspaceHeights(totalPanelHeight - kanbanHeight)
	bounds := []panelBounds{
		{panel: panelExplorer, x: 0, y: top, w: width, h: hostHeight},
		{panel: panelPreview, x: 0, y: top + hostHeight + 2, w: width, h: previewHeight},
	}
	if kanbanHeight > 0 {
		bounds = append(bounds, panelBounds{panel: panelKanban, x: 0, y: top + hostHeight + 2 + previewHeight + 2, w: width, h: kanbanHeight})
	}
	return bounds
}

func workspacePanelBounds(left int, top int, width int, height int) []panelBounds {
	hostsHeight, previewHeight := splitWorkspaceHeights(height)
	return []panelBounds{
		{panel: panelExplorer, x: left, y: top, w: width, h: hostsHeight},
		{panel: panelPreview, x: left, y: top + hostsHeight, w: width, h: previewHeight},
	}
}

func reviewPanelBounds(width int, top int, height int) []panelBounds {
	if width >= 140 {
		queueWidth := threeColumnSideWidth(width)
		activityWidth := threeColumnActivityWidth(width, queueWidth)
		previewWidth := maxInt(60, width-queueWidth-activityWidth-4)
		return []panelBounds{
			{panel: panelReviewQueue, x: 0, y: top, w: queueWidth, h: height},
			{panel: panelPreview, x: queueWidth + 2, y: top, w: previewWidth, h: height},
			{panel: panelActivity, x: queueWidth + 2 + previewWidth + 2, y: top, w: activityWidth, h: height},
		}
	}

	if width >= 104 {
		queueWidth := twoColumnSideWidth(width)
		previewWidth := maxInt(60, width-queueWidth-2)
		return []panelBounds{
			{panel: panelReviewQueue, x: 0, y: top, w: queueWidth, h: height},
			{panel: panelPreview, x: queueWidth + 2, y: top, w: previewWidth, h: height},
		}
	}

	queueHeight := minInt(12, maxInt(8, height/3))
	previewHeight := maxInt(8, height-queueHeight-2)
	return []panelBounds{
		{panel: panelReviewQueue, x: 0, y: top, w: width, h: queueHeight},
		{panel: panelPreview, x: 0, y: top + queueHeight, w: width, h: previewHeight},
	}
}

func focusPanelLabel(panel focusedPanel) string {
	switch panel {
	case panelExplorer:
		return "Tmux Explorer"
	case panelPreview:
		return "Terminal Preview"
	case panelKanban:
		return "Session Board"
	case panelReviewQueue:
		return "Review Queue"
	case panelActivity:
		return "Agent Activity"
	default:
		return ""
	}
}
