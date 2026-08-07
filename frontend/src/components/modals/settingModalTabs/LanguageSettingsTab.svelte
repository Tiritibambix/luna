<script lang="ts">
  import type { Settings } from "../../../lib/client/data/settings.svelte";
  import type { Option } from "../../../types/options";
  import { UserSettingKeys } from "../../../types/settings";
  import SelectInput from "../../forms/SelectInput.svelte";
  import { t } from "@sveltia/i18n";
  import { loadLanguage } from "$lib/common/i18n";

  interface Props {
    settings: Settings;
    languages: Option<string>[];
  }

  let {
    settings,
    languages,
  }: Props = $props();
</script>

<SelectInput
  name={UserSettingKeys.Language}
  placeholder={t("settings.language.display")}
  bind:value={settings.userSettings[UserSettingKeys.Language]}
  options={languages}
  click={(l) => { loadLanguage(l) }}
/>