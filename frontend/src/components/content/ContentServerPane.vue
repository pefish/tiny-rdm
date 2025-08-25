<script setup>
import AddLink from '@/components/icons/AddLink.vue'
import { find } from 'lodash'
import { NButton, useThemeVars } from 'naive-ui'
import useDialogStore from 'stores/dialog.js'
import usePreferencesStore from 'stores/preferences.js'
import { computed } from 'vue'
import { BrowserOpenURL } from 'wailsjs/runtime/runtime.js'

const themeVars = useThemeVars()
const dialogStore = useDialogStore()
const prefStore = usePreferencesStore()

const openBanner = (link) => {
    BrowserOpenURL(link)
}

const skipBanner = () => {
    // Show again after 30 days
    localStorage.setItem('banner_next_time', Date.now() + 30 * 24 * 60 * 60 * 1000)
}

const banner = computed(() => {
    try {
        const nextTime = localStorage.getItem('banner_next_time') || 0
        if (nextTime > 0 && nextTime > Date.now()) {
            return null
        }

        const content = localStorage.getItem('banner')
        const banners = JSON.parse(content)
        let banner = find(banners, ({ lang }) => {
            return lang === prefStore.currentLanguage
        })
        if (banner == null) {
            banner = find(banners, ({ lang }) => {
                return lang === 'en'
            })
        }
        return banner || null
        // return {
        //     lang: 'zh',
        //     title: 'title',
        //     content: 'content',
        //     button: 'button',
        //     link: 'https://redis.tinycraft.cc',
        // }
    } catch {
        return null
    }
})
</script>

<template>
    <div class="content-container flex-box-v">
        <n-alert
            v-if="banner != null"
            :bordered="false"
            :on-close="skipBanner"
            :title="banner.title"
            class="banner"
            closable
            type="warning">
            <span style="margin: 0 10px 0 0">{{ banner.content }}</span>
            <n-button size="small" tertiary type="warning" @click="openBanner(banner.link)">
                {{ banner.button }}
            </n-button>
        </n-alert>

        <!-- TODO: replace icon to app icon -->
        <n-empty :description="$t('interface.empty_server_content')">
            <template #extra>
                <n-button :focusable="false" @click="dialogStore.openNewDialog()">
                    <template #icon>
                        <n-icon :component="AddLink" size="18" />
                    </template>
                    {{ $t('interface.new_conn') }}
                </n-button>
            </template>
        </n-empty>
    </div>
</template>

<style lang="scss" scoped>
@use '@/styles/content';

.content-container {
    justify-content: center;
    padding: 5px;
    box-sizing: border-box;
    position: relative;

    & > .banner {
        position: absolute;
        top: 0;
        left: 0;
        width: 100%;
    }

    & > .sponsor-ad {
        text-align: center;
        margin-top: 20px;
        vertical-align: bottom;
        color: v-bind('themeVars.textColor3');
    }
}

.color-preset-item {
    width: 24px;
    height: 24px;
    margin-right: 2px;
    border: white 3px solid;
    cursor: pointer;

    &_selected,
    &:hover {
        border-color: #cdd0d6;
    }
}
</style>
