//go:build desktop && !windows && !darwin

#include <gtk/gtk.h>
#include <stdint.h>

extern void goMenuInvoke(int id);

static GtkStatusIcon *appTrayIcon;
static GtkWidget *appTrayMenu;

static void appGtkTrayActivate(GtkMenuItem *item, gpointer data) {
    goMenuInvoke((int)(intptr_t)data);
}

static void appGtkTrayPopup(GtkStatusIcon *icon, guint button, guint32 time, gpointer user) {
    (void)icon;
    (void)user;
    if (appTrayMenu) {
        gtk_menu_popup_at_pointer(GTK_MENU(appTrayMenu), NULL);
        (void)button;
        (void)time;
    }
}

void appGtkTrayInstall(const char *title, const char *tooltip) {
    appTrayIcon = gtk_status_icon_new_from_icon_name("application-x-executable");
    if (title && title[0]) {
        gtk_status_icon_set_title(appTrayIcon, title);
    }
    if (tooltip && tooltip[0]) {
        gtk_status_icon_set_tooltip_text(appTrayIcon, tooltip);
    }
    gtk_status_icon_set_visible(appTrayIcon, TRUE);
    appTrayMenu = gtk_menu_new();
    g_signal_connect(appTrayIcon, "popup-menu", G_CALLBACK(appGtkTrayPopup), NULL);
}

void appGtkTrayAddItem(int id, const char *title) {
    GtkWidget *item = gtk_menu_item_new_with_label(title ? title : "");
    g_signal_connect(item, "activate", G_CALLBACK(appGtkTrayActivate), (gpointer)(intptr_t)id);
    gtk_menu_shell_append(GTK_MENU_SHELL(appTrayMenu), item);
}

void appGtkTrayAddSeparator(void) {
    gtk_menu_shell_append(GTK_MENU_SHELL(appTrayMenu), gtk_separator_menu_item_new());
}

void appGtkTrayFinish(void) {
    if (appTrayMenu) gtk_widget_show_all(appTrayMenu);
}
