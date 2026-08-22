// Popup lifecycle events share names with dialogs and sheets (`open` /
// `close`). They must not bubble: choosing a select option would otherwise
// look like the sheet dismissed itself. Listeners on the popup host still
// fire — emit dispatches on the host regardless of bubbles.
export const popupEvent = { bubbles: false };
