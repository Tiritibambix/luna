import { browser } from "$app/environment";
import { register, init, getLocaleFromNavigator, locales, locale, waitLocale } from "@sveltia/i18n";
import { parse } from "yaml";

const languages = [ "en-US", "de-DE", "pl-PL" ];

languages.forEach(x => register(x, () => import(`../../lang/${x}.yaml?raw`).then(m => parse(m.default))));
register("en-DE", () => import(`../../lang/en-US.yaml?raw`).then(m => parse(m.default))); // This is a cheat to get a DD/MM/YYYY format with English. It will be removed once a better method is developed.

init({ fallbackLocale: "en-DE" });

export async function loadLanguage(userChoice: string | null | undefined) {
  await locale.set(await getCurrentLanguage(userChoice));
  await waitLocale("en-DE");
  await waitLocale();
}

export async function getCurrentLanguage(userChoice: string | null | undefined) {
  if (!userChoice || !locales.includes(userChoice)) return getDefaultLanguage();
  return userChoice;
}

export async function getDefaultLanguage() {
  return (browser ? getLocaleFromNavigator() : null) ?? "en-DE";
}