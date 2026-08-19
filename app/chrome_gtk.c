//go:build desktop && !windows && !darwin

#include <gtk/gtk.h>
#include <gdk/gdk.h>

void appWindowMinimize(void *win) {
    if (win) gtk_window_iconify(GTK_WINDOW(win));
}

void appWindowMaximize(void *win) {
    if (win) gtk_window_maximize(GTK_WINDOW(win));
}

void appWindowUnmaximize(void *win) {
    if (win) gtk_window_unmaximize(GTK_WINDOW(win));
}

void appWindowFullscreen(void *win) {
    if (win) gtk_window_fullscreen(GTK_WINDOW(win));
}

void appWindowUnfullscreen(void *win) {
    if (win) gtk_window_unfullscreen(GTK_WINDOW(win));
}

void appWindowShow(void *win) {
    if (win) gtk_widget_show(GTK_WIDGET(win));
}

void appWindowHide(void *win) {
    if (win) gtk_widget_hide(GTK_WIDGET(win));
}

void appWindowFocus(void *win) {
    if (win) gtk_window_present(GTK_WINDOW(win));
}

void appWindowSetAlwaysOnTop(void *win, int on) {
    if (win) gtk_window_set_keep_above(GTK_WINDOW(win), on ? TRUE : FALSE);
}

void appWindowSetDecorations(void *win, int on) {
    if (win) gtk_window_set_decorated(GTK_WINDOW(win), on ? TRUE : FALSE);
}

// GTK has no transparent title bar to draw behind, so inset and hidden are
// the same thing here: the decoration goes and the page draws its own bar,
// buttons included. Saying so in one place is better than pretending the two
// differ and leaving a caller to find out they do not.
void appWindowSetTitlebar(void *win, int style) {
    if (!win) return;
    gtk_window_set_decorated(GTK_WINDOW(win), style == 0);
}

void appWindowBeginDrag(void *win) {
    if (!win) return;
    GdkWindow *gw = gtk_widget_get_window(GTK_WIDGET(win));
    if (!gw) return;
    GdkDisplay *display = gdk_window_get_display(gw);
    GdkSeat *seat = gdk_display_get_default_seat(display);
    GdkDevice *pointer = seat ? gdk_seat_get_pointer(seat) : NULL;
    if (!pointer) return;

    // The window manager runs the move, so it needs where the pointer is and
    // which button is holding it. Asking the device beats threading the
    // originating event down from Go.
    int rx = 0, ry = 0;
    GdkModifierType mask;
    gdk_device_get_position(pointer, NULL, &rx, &ry);
    gdk_window_get_device_position(gw, pointer, NULL, NULL, &mask);
    if (!(mask & GDK_BUTTON1_MASK)) return;

    gtk_window_begin_move_drag(GTK_WINDOW(win), 1, rx, ry, GDK_CURRENT_TIME);
}

void appWindowSetPosition(void *win, int x, int y) {
    if (win) gtk_window_move(GTK_WINDOW(win), x, y);
}

void appWindowPosition(void *win, int *x, int *y) {
    if (!win || !x || !y) return;
    gtk_window_get_position(GTK_WINDOW(win), x, y);
}

void appWindowCenter(void *win) {
    if (!win) return;
    gtk_window_set_position(GTK_WINDOW(win), GTK_WIN_POS_CENTER);
}
