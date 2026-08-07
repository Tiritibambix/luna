import { browser } from "$app/environment";
import { register, init, getLocaleFromNavigator, locales, locale, waitLocale } from "@sveltia/i18n";
import { parse } from "yaml";

const languages = [ "en-US", "en-DE", "de-DE", "pl-PL" ];

languages.forEach(x => register(x, () => import(`../../lang/${x}.yaml?raw`).then(m => parse(m.default))));

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