<template>
  <section class="dashboard content">
    <header class="columns">
      <div class="column is-two-thirds">
        <h1 class="title is-5">
          {{ $utils.niceDate(new Date()) }}
        </h1>
      </div>
    </header>

    <section class="counts wrap">
      <div class="tile is-ancestor">
        <div class="tile is-vertical is-12">
          <div class="tile">
            <div class="tile is-parent is-vertical relative">
              <b-loading v-if="isCountsLoading" active :is-full-page="false" />
              <article class="tile is-child notification" data-cy="lists">
                <div class="columns is-mobile">
                  <div class="column is-6">
                    <p class="title">
                      <b-icon icon="format-list-bulleted-square" />
                      {{ $utils.niceNumber(counts.lists.total) }}
                    </p>
                    <p class="is-size-6 has-text-grey">
                      {{ $tc('globals.terms.list', counts.lists.total) }}
                    </p>
                  </div>
                  <div class="column is-6">
                    <ul class="no has-text-grey">
                      <li>
                        <label for="#">{{ $utils.niceNumber(counts.lists.public) }}</label>
                        {{ $t('lists.types.public') }}
                      </li>
                      <li>
                        <label for="#">{{ $utils.niceNumber(counts.lists.private) }}</label>
                        {{ $t('lists.types.private') }}
                      </li>
                      <li>
                        <label for="#">{{ $utils.niceNumber(counts.lists.optinSingle) }}</label>
                        {{ $t('lists.optins.single') }}
                      </li>
                      <li>
                        <label for="#">{{ $utils.niceNumber(counts.lists.optinDouble) }}</label>
                        {{ $t('lists.optins.double') }}
                      </li>
                    </ul>
                  </div>
                </div>
              </article><!-- lists -->

              <article class="tile is-child notification" data-cy="campaigns">
                <div class="columns is-mobile">
                  <div class="column is-6">
                    <p class="title">
                      <b-icon icon="rocket-launch-outline" />
                      {{ $utils.niceNumber(counts.campaigns.total) }}
                    </p>
                    <p class="is-size-6 has-text-grey">
                      {{ $tc('globals.terms.campaign', counts.campaigns.total) }}
                    </p>
                  </div>
                  <div class="column is-6">
                    <ul class="no has-text-grey">
                      <li v-for="(num, status) in counts.campaigns.byStatus" :key="status">
                        <label for="#" :data-cy="`campaigns-${status}`">{{ num }}</label>
                        {{ $t(`campaigns.status.${status}`) }}
                        <span v-if="status === 'running'" class="spinner is-tiny">
                          <b-loading :is-full-page="false" active />
                        </span>
                      </li>
                    </ul>
                  </div>
                </div>
              </article><!-- campaigns -->
            </div><!-- block -->

            <div class="tile is-parent relative">
              <b-loading v-if="isCountsLoading" active :is-full-page="false" />
              <article class="tile is-child notification" data-cy="subscribers">
                <div class="columns is-mobile">
                  <div class="column is-6">
                    <p class="title">
                      <b-icon icon="account-multiple" />
                      {{ $utils.niceNumber(counts.subscribers.total) }}
                    </p>
                    <p class="is-size-6 has-text-grey">
                      {{ $tc('globals.terms.subscriber', counts.subscribers.total) }}
                    </p>
                  </div>

                  <div class="column is-6">
                    <ul class="no has-text-grey">
                      <li>
                        <label for="#">{{ $utils.niceNumber(counts.subscribers.blocklisted) }}</label>
                        {{ $t('subscribers.status.blocklisted') }}
                      </li>
                      <li>
                        <label for="#">{{ $utils.niceNumber(counts.subscribers.orphans) }}</label>
                        {{ $t('dashboard.orphanSubs') }}
                      </li>
                    </ul>
                  </div><!-- subscriber breakdown -->
                </div><!-- subscriber columns -->
                <hr />
                <div class="columns" data-cy="messages">
                  <div class="column is-12">
                    <p class="title">
                      <b-icon icon="email-outline" />
                      {{ $utils.niceNumber(counts.messages) }}
                    </p>
                    <p class="is-size-6 has-text-grey">
                      {{ $t('dashboard.messagesSent') }}
                    </p>
                  </div>
                </div>
              </article><!-- subscribers -->
            </div>

            <div class="tile is-parent relative" v-if="webhookStats.length > 0">
              <b-loading v-if="isWebhooksLoading" active :is-full-page="false" />
              <article class="tile is-child notification" data-cy="webhooks">
                <div class="columns is-mobile">
                  <div class="column is-6">
                    <p class="title">
                      <b-icon icon="webhook" />
                      {{ webhookStats.length }}
                    </p>
                    <p class="is-size-6 has-text-grey">
                      {{ $t('dashboard.webhooks') }}
                    </p>
                  </div>
                  <div class="column is-6">
                    <ul class="no has-text-grey">
                      <li>
                        <label for="#">{{ $utils.niceNumber(webhookTotals().dispatched) }}</label>
                        {{ $t('dashboard.webhooksDispatched') }}
                      </li>
                      <li>
                        <label for="#" class="has-text-success">{{ $utils.niceNumber(webhookTotals().success) }}</label>
                        {{ $t('dashboard.webhooksSuccess') }}
                      </li>
                      <li>
                        <label for="#" :class="{ 'has-text-danger': webhookTotals().failed > 0 }">
                          {{ $utils.niceNumber(webhookTotals().failed) }}
                        </label>
                        {{ $t('dashboard.webhooksFailed') }}
                      </li>
                    </ul>
                  </div>
                </div>
                <hr />
                <div class="webhook-endpoints">
                  <div v-for="stat in webhookStats" :key="stat.name" class="columns is-mobile is-size-7">
                    <div class="column is-6">
                      <strong>{{ stat.name }}</strong>
                    </div>
                    <div class="column is-3 has-text-right">
                      <span :class="successRateClass(stat)">
                        {{ successRate(stat) }}%
                      </span>
                      ({{ stat.totalSuccess }}/{{ stat.totalDispatched }})
                    </div>
                    <div class="column is-3 has-text-right has-text-grey">
                      {{ formatRelativeTime(stat.lastDispatch) }}
                    </div>
                  </div>
                </div>
                <div v-if="webhookTotals().failed > 0" class="has-text-danger is-size-7 mt-2">
                  <b-icon icon="alert-circle-outline" size="is-small" />
                  {{ webhookStats.find(s => s.lastError)?.lastError || '' }}
                </div>
              </article><!-- webhooks -->
            </div>
          </div>
          <div class="tile is-parent relative">
            <b-loading v-if="isChartsLoading" active :is-full-page="false" />
            <article class="tile is-child notification charts">
              <div class="columns">
                <div class="column is-6">
                  <h3 class="title is-size-6">
                    {{ $t('dashboard.campaignViews') }}
                  </h3><br />
                  <chart type="line" v-if="campaignViews" :data="campaignViews" />
                </div>
                <div class="column is-6">
                  <h3 class="title is-size-6 has-text-right">
                    {{ $t('dashboard.linkClicks') }}
                  </h3><br />
                  <chart type="line" v-if="campaignClicks" :data="campaignClicks" />
                </div>
              </div>
            </article>
          </div>
        </div>
      </div><!-- tile block -->
      <p v-if="settings['app.cache_slow_queries']" class="has-text-grey">
        *{{ $t('globals.messages.slowQueriesCached') }}
        <a href="https://listmonk.app/docs/maintenance/performance/" target="_blank" rel="noopener noreferer"
          class="has-text-grey">
          <b-icon icon="link-variant" /> {{ $t('globals.buttons.learnMore') }}
        </a>
      </p>
    </section>
  </section>
</template>

<script>
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import Vue from 'vue';
import { mapState } from 'vuex';
import { colors } from '../constants';
import Chart from '../components/Chart.vue';

dayjs.extend(relativeTime);

export default Vue.extend({
  components: {
    Chart,
  },

  data() {
    return {
      isChartsLoading: true,
      isCountsLoading: true,
      isWebhooksLoading: true,
      campaignViews: null,
      campaignClicks: null,
      counts: {
        lists: {},
        subscribers: {},
        campaigns: {},
        messages: 0,
      },
      webhookStats: [],
    };
  },

  methods: {
    makeChart(data) {
      if (data.length === 0) {
        return {};
      }
      return {
        labels: data.map((d) => dayjs(d.date).format('DD MMM')),
        datasets: [
          {
            data: [...data.map((d) => d.count)],
            borderColor: colors.primary,
            borderWidth: 2,
            pointHoverBorderWidth: 5,
            pointBorderWidth: 0.5,
          },
        ],
      };
    },

    webhookTotals() {
      return this.webhookStats.reduce(
        (acc, s) => ({
          dispatched: acc.dispatched + s.totalDispatched,
          success: acc.success + s.totalSuccess,
          failed: acc.failed + s.totalFailed,
        }),
        { dispatched: 0, success: 0, failed: 0 },
      );
    },

    formatRelativeTime(timestamp) {
      if (!timestamp || timestamp === '0001-01-01T00:00:00Z') {
        return '-';
      }
      return dayjs(timestamp).fromNow();
    },

    successRate(stat) {
      if (stat.totalDispatched === 0) return 0;
      return Math.round((stat.totalSuccess / stat.totalDispatched) * 100);
    },

    successRateClass(stat) {
      const rate = this.successRate(stat);
      return {
        'has-text-success': rate >= 90,
        'has-text-warning': rate >= 50 && rate < 90,
        'has-text-danger': rate < 50 && stat.totalDispatched > 0,
      };
    },
  },

  computed: {
    ...mapState(['settings']),
    dayjs() {
      return dayjs;
    },
  },

  mounted() {
    // Pull the counts.
    this.$api.getDashboardCounts().then((data) => {
      this.counts = data;
      this.isCountsLoading = false;
    });

    // Pull the charts.
    this.$api.getDashboardCharts().then((data) => {
      this.isChartsLoading = false;
      this.campaignViews = this.makeChart(data.campaignViews);
      this.campaignClicks = this.makeChart(data.linkClicks);
    });

    // Pull webhook stats.
    this.$api.getWebhookStats().then((data) => {
      this.webhookStats = data || [];
      this.isWebhooksLoading = false;
    });
  },
});
</script>
