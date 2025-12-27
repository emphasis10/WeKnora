import { createApp } from "vue";
import { createPinia } from "pinia";
import App from "./App.vue";
import router from "./router";
import "./assets/fonts.css";
import TDesign from "tdesign-vue-next";
// Import a few global style variables from the component library
import "tdesign-vue-next/es/style/index.css";
import "@/assets/theme/theme.css";
import i18n from "./i18n";

const app = createApp(App);

app.use(TDesign);
app.use(createPinia());
app.use(router);
app.use(i18n);

app.mount("#app");
