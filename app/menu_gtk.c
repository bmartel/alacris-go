//go:build desktop && !windows && !darwin

#include <gtk/gtk.h>
#include <gdk/gdk.h>
#include <stdint.h>
#include <string.h>

extern void goMenuInvoke(int id);

static GtkWidget *menuBar;
static GtkWidget *menuStack[16];
static int menuDepth;
static int menuAttached;

static void appGtkActivate(GtkMenuItem *item, gpointer data) {
    goMenuInvoke((int)(intptr_t)data);
}

void appGtkInstallMenu(void *win) {
    GtkWindow *window = GTK_WINDOW(win);
    GtkWidget *child = gtk_bin_get_child(GTK_BIN(window));
    menuBar = gtk_menu_bar_new();
    menuDepth = 0;
    menuStack[0] = menuBar;
    if (!menuAttached && child) {
        g_object_ref(child);
        gtk_container_remove(GTK_CONTAINER(window), child);
        GtkWidget *vbox = gtk_box_new(GTK_ORIENTATION_VERTICAL, 0);
        gtk_box_pack_start(GTK_BOX(vbox), menuBar, FALSE, FALSE, 0);
        gtk_box_pack_start(GTK_BOX(vbox), child, TRUE, TRUE, 0);
        g_object_unref(child);
        gtk_container_add(GTK_CONTAINER(window), vbox);
        gtk_widget_show_all(vbox);
        menuAttached = 1;
    }
}

static GtkWidget *appGtkCurrent(void) {
    return menuStack[menuDepth];
}

void appGtkBeginSubmenu(const char *title) {
    GtkWidget *item = gtk_menu_item_new_with_label(title ? title : "");
    GtkWidget *sub = gtk_menu_new();
    gtk_menu_item_set_submenu(GTK_MENU_ITEM(item), sub);
    gtk_menu_shell_append(GTK_MENU_SHELL(appGtkCurrent()), item);
    if (menuDepth < 15) {
        menuDepth++;
        menuStack[menuDepth] = sub;
    }
}

void appGtkEndSubmenu(void) {
    if (menuDepth > 0) {
        menuDepth--;
    }
}

void appGtkAddSeparator(void) {
    gtk_menu_shell_append(GTK_MENU_SHELL(appGtkCurrent()), gtk_separator_menu_item_new());
}

void appGtkAddMenuItem(int id, const char *title, const char *key, int shift, int alt, int ctrl) {
    GtkWidget *item = gtk_menu_item_new_with_label(title ? title : "");
    g_signal_connect(item, "activate", G_CALLBACK(appGtkActivate), (gpointer)(intptr_t)id);
    gtk_menu_shell_append(GTK_MENU_SHELL(appGtkCurrent()), item);
    (void)key;
    (void)shift;
    (void)alt;
    (void)ctrl;
}

void appGtkFinishMenu(void) {
    if (menuBar) {
        gtk_widget_show_all(menuBar);
    }
}
