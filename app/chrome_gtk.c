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
