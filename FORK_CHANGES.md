# 二次开发变更记录（Fork Changes）

本仓库 fork 自上游开源项目 new-api（QuantumNous），在其之上做二次开发。
本文档记录**所有对上游代码的改动**，以及这些改动背后的需求与约束。

---

## 目录

**要动手操作，直接看这两节：**

- 🔄 **官方发布新版本了** → [五、同步上游更新的操作流程](#五同步上游更新的操作流程)
- 🛠️ **要改代码 / 加功能** → [六、功能分支的操作流程](#六功能分支的操作流程)

**其余：**

- [一、本文档的维护约定](#一本文档的维护约定)
- [二、仓库与分支约定](#二仓库与分支约定) — 三个分支各自的用途与流向图
- [三、贯穿所有改动的核心约束](#三贯穿所有改动的核心约束)
- [四、变更记录](#四变更记录) — 改过什么、改在哪，**解冲突时对照这里**
- [七、记录模板](#七记录模板)

> 相关文件：`.agents/rules/fork-changes.md` 是给 AI 助手看的强制规则（精简版），
> 由 `CLAUDE.md` 末尾一行指向它；本文档是给人看的完整版。

---

## 一、本文档的维护约定

- **每一次修改代码，都必须在「四、变更记录」中追加一条记录**，不得只提交代码而不记录。
- 一条记录 = 一个功能 / 一次修复，对应一个功能分支（通常也对应一个 PR）。
- 记录必须写清楚 **新增了哪些文件**、**改动了哪些上游既有文件以及各自改了什么**。
  这一段是同步上游新版本、解决合并冲突时的主要依据。
- 文档放在仓库根目录而不是 `docs/`：`AGENTS.md` 规定未经明确要求不得在 `docs/` 下新增文件；
  同时根目录的新文件不会与上游更新产生冲突。
- 追加新记录时使用「七、记录模板」。
- 官方发布新版本时，按「五、同步上游更新的操作流程」执行；开发新功能时，按「六、功能分支的操作流程」执行。
- 强制这条约定的规则写在 `.agents/rules/fork-changes.md`，由 `CLAUDE.md` 末尾的一行指向它
  —— 这一行是让规则在新会话里自动生效所必需的最小改动。

---

## 二、仓库与分支约定

| 分支 | 用途 | 从哪来 | 合到哪去 |
| --- | --- | --- | --- |
| `main` | 上游官方仓库的**纯镜像**，只同步，不改业务代码 | 上游 `upstream/main`（只快进） | `production` |
| `production` | 发布分支，官方代码 + 我们的改动，部署以此为准 | `main` 与功能分支 | —（终点） |
| 功能分支 | 一个功能 / 一次修复 | **从 `production` 切出** | `production` |

两条线都汇入 `production`，但**各走各的，不要混在一次合并里**：

```
上游 upstream/main ──快进──> main ──同步合并──┐
                                              ├──> production ──> 部署
          功能分支（从 production 切出）──────┘
```

- **同步线**（`上游 → main → production`）：只带官方的改动。
- **功能线**（`production → 功能分支 → production`）：只带我们自己的改动。

**功能分支必须从 `production` 切出，不要从 `main` 切出。**
从 `main` 切出会导致合并时把 `main` 上那些官方提交一起带进 `production` ——
一次合并同时承载「新功能」和「上游同步」两种变化，PR 的 diff 里混着官方提交，
出问题也分不清是谁引起的。从 `production` 切出，功能分支的 diff 就只有我们自己写的东西。

代价是：`production` 可能落后于最新官方代码。所以**开发新功能前，先按第五节把
`production` 同步到最新**，再切分支，这样既基于最新官方代码，又保持两条线分离。

`production` 最初与 `main` 完全一致，只有修改了开源项目代码之后，才会把改动合并进
`production`，再发布部署。

### GitHub / 协作要求

- 开发必须在指定的功能分支上进行，**不得直接推送到其他分支**。
- 提交信息要清晰描述改动内容与原因。
- 推送使用 `git push -u origin <branch-name>`；网络失败时按 2s / 4s / 8s / 16s 退避重试，最多 4 次。
- **除非明确要求，否则不主动创建 PR**。创建 PR 时若仓库存在 PR 模板，则按模板结构填写。
- 如果某个功能分支对应的 PR 已经合并，后续工作视为全新改动：从最新的默认分支重新拉出同名分支，
  不要在已合并的历史之上继续堆提交。

---

## 三、贯穿所有改动的核心约束

1. **可持续跟随上游更新（最重要）**
   改动必须做到：即使开源项目发布新版本，也能顺利把上游代码合并进来，不会因为我们改过代码而合不了。
   具体做法：
   - 功能逻辑尽量放进**全新文件**，新文件与上游永远不冲突；
   - 对上游既有文件只做**追加式的最小改动**（加一行注册、加一个 `case`、加一个分支），
     不重排、不重构、不动无关代码行；
   - 不为了统一风格去修改上游既有代码（包括既存的 lint 告警）。
2. 遵守 `AGENTS.md`、`web/AGENTS.md`、`.agents/rules/billing.md` 中的项目规范。
3. **受保护信息不得改动**：项目名 new-api、组织/作者 QuantumNous 相关的所有品牌、署名、
   元数据、模块路径等一律保留。
4. 数据库改动必须同时兼容 SQLite / MySQL / PostgreSQL。

---

## 四、变更记录

### 需求来源（首次沟通，2026-09-21）

用户原话（保持原样，不做改写）：

> https://payerscan.com/
> https://docs.payerscan.com/  ；new api 是一个开源项目，我已经拉取到我的仓库，我已经成功部署它，
> 但是我需要把我的支付方式对接到项目，给你的两个连接就是我需要对接的支付接口，一个是官网一个文档链接；
> 现在main分支是fork 了new api 官网用于同步官方最新版本，仓库还有一个production分支，用于修改代码后
> 同步到production然后正式发布；所以你改代码建立分支要考虑到代码可以很好的合并到分支production代码，
> production代码刚开始是跟main分支代码一致的，只有修改了开源项目代码才会合并到production然后发布部署；
> 总之你改的代码能干很好的合并到开源项目，不能因为开源项目更新版本了无法合并你改的代码

后续补充要求：

> 1.每次修改用文档记录；2.把我第一次跟你沟通改代码与github要求等也记录到文档

---

### 001 — 接入 PayerScan 加密货币支付网关

| 项 | 内容 |
| --- | --- |
| 日期 | 2026-09-21 |
| 分支 | `claude/zealous-babbage-x30yah` |
| PR | https://github.com/qiin/new-api/pull/1 |
| 提交 | `628b980` |
| 基线 | `972aed1`（`main`） |

> 注：本分支是从 `main` 切出的（当时 `production` 还没有任何自有改动，树与 `main` 的祖先一致），
> 因此合并进 `production` 是 fast-forward，会把 `main` 上那 8 个官方提交一起带进去 —— 这次是干净的。
> 从 002 开始按第二节的规定，功能分支一律从 `production` 切出。

#### 需求

把 PayerScan（加密货币收款网关）接入充值流程，与现有的易支付 / Stripe / Creem / Waffo
并列成为一个独立的充值渠道。

参考资料：官网 https://payerscan.com/ ，文档 https://docs.payerscan.com/

#### 对接的 PayerScan 接口

| 用途 | 接口 |
| --- | --- |
| 创建收款发票 | `POST https://api.payerscan.com/payment/crypto` |
| 查询发票状态 | `GET  https://api.payerscan.com/invoice/:trans_id` |
| 鉴权 | 请求头 `x-api-key: <店铺 API Key>` |
| 回调 | PayerScan `POST` 到我们的 `callback_url`，事件只有 `completed` 与 `expired` |

发票状态：`waiting` / `processing` / `completed` / `expired`。

#### 新增文件（与上游零冲突）

| 文件 | 作用 |
| --- | --- |
| `setting/payment_payerscan.go` | 配置项变量 |
| `service/payerscan.go` | PayerScan API 客户端：建单、查单、基址解析与校验 |
| `controller/topup_payerscan.go` | 下单、金额试算、回调处理、渠道可用性判断 |
| `model/topup_payerscan.go` | 渠道常量 + 入账事务 `RechargePayerScan` |
| `controller/topup_payerscan_test.go` | 后端测试（集中在一个文件，不跨层散落） |
| `web/src/features/wallet/hooks/use-payerscan-payment.ts` | 前端支付 hook |
| `web/src/features/system-settings/integrations/payerscan-settings-section.tsx` | 管理端独立配置分区 |

渠道可用性判断函数（`isPayerScanTopUpEnabled` 等）特意放在 `controller/topup_payerscan.go`
里，因此上游的 `controller/payment_webhook_availability.go` **一行未动**。

管理端也特意**没有改动 1636 行的 `payment-settings-section.tsx`**，而是新建独立分区，
只在 section 注册表里加一项。

#### 修改的上游既有文件（全部为追加式改动）

| 文件 | 改了什么 |
| --- | --- |
| `model/option.go` | `OptionMap` 注册 6 个 `PayerScan*` 键；`updateOptionMap` 的 switch 增加 6 个 `case` |
| `router/api-router.go` | 新增 3 条路由（见下） |
| `controller/topup.go` | import 增加 `slices`；`GetTopUpInfo` 追加 payerscan 支付方式条目、`enable_payerscan_topup`、`payerscan_min_topup` |
| `controller/option.go` | 在既有的 `TaskPublicAddress` 校验块之后，追加 `PayerScanBaseURL` 的 URL 校验 |
| `web/src/features/wallet/constants.ts` | `PAYMENT_TYPES` 与 `PAYMENT_ICON_COLORS` 各加一项 |
| `web/src/features/wallet/types.ts` | 新增 PayerScan 请求/响应类型；`TopupInfo` 加 2 个字段 |
| `web/src/features/wallet/api.ts` | 新增 2 个 API 函数 |
| `web/src/features/wallet/lib/payment.ts` | 新增 `isPayerScanPayment`；调度、默认渠道、最低充值额各加一个分支 |
| `web/src/features/wallet/lib/ui.tsx` | 图标 switch 加一个 `case` |
| `web/src/features/wallet/hooks/index.ts` | 导出新 hook |
| `web/src/features/wallet/hooks/use-payment.ts` | 金额试算加一个分支 |
| `web/src/features/wallet/index.tsx` | 接入 hook、调度与 loading 状态 |
| `web/src/features/wallet/components/recharge-form-card.tsx` | 新增 `enablePayerScanTopup` prop |
| `web/src/features/wallet/lib/payment.test.ts` | 补 PayerScan 用例并补齐新增的必填 stub |
| `web/src/features/wallet/hooks/use-payment.test.ts` | 同上 |
| `web/src/features/system-settings/types.ts` | `BillingSettings` 加 5 个字段 |
| `web/src/features/system-settings/billing/index.tsx` | 默认值加 5 项 |
| `web/src/features/system-settings/billing/section-registry.tsx` | 注册 `payerscan` 分区 + 回调地址推导函数 |
| `web/src/i18n/locales/*.json`（7 个语言） | 新增 14 条文案 |

#### 新增路由

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `POST` | `/api/payerscan/webhook` | PayerScan 回调（匿名，带请求体大小限制） |
| `POST` | `/api/user/payerscan/amount` | 金额试算（登录态） |
| `POST` | `/api/user/payerscan/pay` | 发起支付（登录态，带关键操作限流） |

#### 新增配置项

| 键 | 说明 |
| --- | --- |
| `PayerScanEnabled` | 渠道开关 |
| `PayerScanMerchantID` | 商户号（`MID-XXXXXXXXXX`） |
| `PayerScanApiKey` | 店铺 API Key。键名以 `Key` 结尾，会被 `GET /api/option/` 自动屏蔽，不回传前端 |
| `PayerScanBaseURL` | API 基址，留空使用生产地址 `https://api.payerscan.com`；保存时校验必须是 http/https |
| `PayerScanUnitPrice` | 每个充值单位收取的美元金额 |
| `PayerScanMinTopUp` | 最低充值数量 |

#### 安全设计（重点）

PayerScan 的回调**没有签名机制**：`completed` 事件在 body 里回传 `api_key`，
`expired` 事件连 `api_key` 都没有。因此回调处理采用多层防护：

1. `merchant_id` 与 `api_key` 使用 `subtle.ConstantTimeCompare` 做常量时间比对；
2. 通过后**先查本地订单**是否存在、是否属于 PayerScan、是否处于待支付状态
   —— 因为 `expired` 事件合法地不带 API Key，没有这一步，未鉴权的请求就能让网关按需对外发起 HTTP 请求；
3. 再回查 `GET /invoice/:trans_id`，**以上游返回的状态为准**，并校验发票的 `request_id`
   与回调一致、发票状态与回调状态一致 —— 伪造回调无法入账；
4. 入账在单个数据库事务内完成：行锁（`lockForUpdate`）→ 渠道校验 → 状态校验 →
   **实付金额不低于订单金额**（容差 1 分，覆盖 PayerScan 换算的分位取整）→ 标记完成 → 增加额度。
   重复 / 并发回调（含多实例部署）最多入账一次；
5. 回调 body **不写日志**（含 API Key）；
6. 下单前校验回调地址已配置，避免收了钱却没有回调入口；
7. 回查失败返回 `500` 让 PayerScan 重试，永久性失败返回 `200` 避免无意义重试。

额度换算统一走 `common.WalletQuotaFromDecimalStrict`（`.agents/rules/billing.md` 的要求）。

#### 管理员配置步骤

1. 在 PayerScan 后台创建 Store，配置至少一个收款方式（钱包地址或 Binance Pay），拿到商户号与 API Key。
2. new-api 后台：系统设置 → 计费 → **PayerScan（加密货币）**，填入商户号、API Key、单价、最低充值量，
   打开开关保存。
3. 页面会显示需要登记到 PayerScan 店铺的回调地址 `<回调地址>/api/payerscan/webhook`（带复制按钮），
   把它填到 PayerScan 店铺的 callback URL。
4. 支付渠道需要先完成**支付合规确认**才会对用户展示（与其他渠道一致）。

#### 验证结果

| 检查 | 结果 |
| --- | --- |
| `go build ./...` | 通过 |
| `go vet ./controller/... ./service/... ./model/... ./setting/... ./router/...` | 通过 |
| `go test ./controller/... ./model/... ./setting/...` | 通过（新增 6 个测试） |
| `bun run typecheck` | 通过 |
| `bun run test` | 154 个文件 / 1949 个用例全部通过 |
| `bun run build` | 通过 |
| `oxlint`（本次改动的文件） | 无 error |

新增测试覆盖：渠道可用性（开关 / 凭证 / 合规确认）、金额换算（分组倍率 / 折扣 / tokens 展示模式）、
回调鉴权（商户号错误、缺 API Key、API Key 错误、报文损坏、外部订单号、未知状态、渠道关闭）、
建单报文格式、查单报文解析（金额字符串与数字两种形式、错误信封）、基址校验。

#### 上游合并实测

拉取上游 `QuantumNous/new-api` 当时领先的 24 个提交，用 `git merge-tree --write-tree` 实测合并：

- 所有 Go 文件、所有前端 TSX/TS 文件**全部自动合并成功**，包括上游同期也改过的 `model/option.go`。
- 唯一冲突出现在 7 个 i18n 语言包：最初把新文案追加在文件末尾，而末尾正是上游放新文案的区域，
  每个文件撞出 23 行冲突。
- **修正**：把 14 条新文案按字母序插进 A-Z 有序区（`"Zoom"` 之前），冲突降到每个文件 2 行
  —— 上游在 `"Enable Passkey"` 后面也插了一条，两边各一行，保留两行即可。
  这类冲突是 6700 条扁平键的 JSON 按行合并的固有现象，任何人加文案都会遇到，2 行是可接受的下限。

此约定已写入 `.agents/rules/fork-changes.md`，后续改动照此执行。

另外确认：`production` 当前落后 `main` 8 个提交且**没有任何自有改动**（树与 `main` 一致），
本分支包含 `production` 的全部历史，合并进 `production` 是 fast-forward，零冲突。

#### 已知限制 / 待办

- **未执行三数据库（SQLite / MySQL / PostgreSQL）验证矩阵**：本次没有改 schema、迁移或 GORM tag，
  复用了既有的 `TopUp` 表结构与 `lockForUpdate`，按 `AGENTS.md` 的口径不构成 schema 变更；
  但环境中没有真实的 MySQL / PostgreSQL 实例，未做实际验证。
- `web/src/features/wallet/lib/ui.tsx:21` 存在一条**上游既有的** lint error
  （`no-import-type-side-effects`），不在本次改动的行上，为减少与上游的冲突面未做修改。
- PayerScan 目前只支持 USD 计价，不支持多币种。
- 下单时若 PayerScan 返回失败，订单会被标记为 `failed`。极端情况下（PayerScan 已建单但响应丢失）
  用户仍可能完成支付，此时回调会命中「已关闭订单收到支付完成回调」的 error 日志，需要人工核对。

### 002 — 新增站点级开关：允许/禁止边栏个性化设置

| 项 | 内容 |
| --- | --- |
| 日期 | 2026-09-21 |
| 分支 | `feature/sidebar-personalization-toggle` |
| PR | （待创建） |
| 提交 | （待提交） |
| 基线 | `1811d6d33`（`production`） |

#### 需求

个人资料页右侧的「左侧边栏个人设置」卡片（`SidebarModulesCard`，用户可自行勾选要在侧边栏
显示哪些区域）此前是否显示完全写死在后端角色逻辑里：超级管理员看不到，管理员和普通用户都
必然能看到，没有站点级开关。用户希望能在管理端把这张卡片对管理员和普通用户也隐藏掉。

#### 修改的上游既有文件（全部为追加式改动）

| 文件 | 改了什么 |
| --- | --- |
| `common/constants.go` | 新增变量 `SidebarPersonalizationEnabled`（默认 `true`），紧跟在既有的 `DefaultCollapseSidebar` 变量之后 |
| `model/option.go` | `InitOptionMap` 追加一行注册该键；`updateOptionMap` 的布尔开关分支追加一个 `case`（键名以 `Enabled` 结尾，自动落入既有的通用布尔分支，未改动那条 `HasSuffix` 判断本身） |
| `controller/user.go` | `calculateUserPermissions` 在超级管理员分支之后追加一个 `else if !common.SidebarPersonalizationEnabled` 分支，管理员与普通用户在开关关闭时也返回 `sidebar_settings=false`；未改动既有的角色分支内部逻辑 |
| `web/src/features/system-settings/types.ts` | `OperationsSettings` 追加字段 `SidebarPersonalizationEnabled: boolean` |
| `web/src/features/system-settings/operations/index.tsx` | 默认值对象追加 `SidebarPersonalizationEnabled: true` |
| `web/src/features/system-settings/operations/section-registry.tsx` | 传给 `SystemBehaviorSection` 的 `defaultValues` 追加该字段 |
| `web/src/features/system-settings/general/system-behavior-section.tsx` | schema 追加字段；在既有的「默认折叠侧边栏」开关之后追加一个新的 `FormField` 开关，UI 结构与其完全一致 |
| `web/src/i18n/locales/*.json`（7 个语言） | 新增 2 条文案，按约定插入 A-Z 有序区，不追加到文件末尾 |

前端「显示与否」的判断逻辑（`web/src/features/profile/index.tsx` 的 `canConfigureSidebar =
permissions?.sidebar_settings !== false`）**完全没有改动** —— 它早已读取后端下发的
`sidebar_settings`，本次只是让后端多一种情况能把这个值算成 `false`。

#### 新增配置项

| 键 | 说明 |
| --- | --- |
| `SidebarPersonalizationEnabled` | 布尔，默认 `true`。关闭后管理员和普通用户都不再看到「左侧边栏个人设置」卡片，超级管理员不受影响（本来就看不到）。位置：系统设置 → 运营 → 系统行为 → 「允许边栏个性化设置」 |

#### 设计要点

- 范围选择（对管理员和普通用户都隐藏，而不是只隐藏普通用户）由用户在对话中明确选定。
- 关闭开关不会清除用户此前保存的个人侧边栏偏好（`sidebar_modules`）：`use-sidebar-config.ts`
  里已有的逻辑是 `sidebar_settings === false` 时跳过用户偏好覆盖层，直接呈现管理员配置的完整
  视图；重新打开开关后旧偏好会自动生效，不需要额外的数据迁移或清理。这与超级管理员分支复用
  同一条既有逻辑，未新增分支。
- 没有新建文件：改动点分散在 8 个上游文件里，但每处都是"加一行 / 加一个 case / 加一个
  FormField 块"，没有重排或重构既有代码。

#### 验证结果

| 检查 | 结果 |
| --- | --- |
| `go build ./...` | 通过 |
| `go vet ./controller/... ./common/... ./model/... ./setting/... ./router/...` | 通过 |
| `go test ./controller/... ./model/... ./setting/... ./common/...` | 通过（新增 1 个测试文件，6 个用例） |
| `bun run typecheck` | 通过 |
| `bun run test` | 154 个文件 / 1949 个用例全部通过 |
| `bun run build` | 通过 |
| `oxlint`（本次改动的文件） | 无 error |

新增测试 `controller/user_permissions_test.go`：覆盖开关开/关状态下超级管理员、管理员、普通
用户三种角色的 `sidebar_settings` 取值，确认超级管理员始终被排除、开关关闭后管理员和普通用户
都被排除、开关打开后二者都恢复可配置。

#### 已知限制 / 待办

- 未做数据库改动，未触发三数据库验证矩阵的适用条件。
- 未新增前端测试：`system-behavior-section.tsx` 里既有的同类开关（`DefaultCollapseSidebar`）
  本身也没有专门的单元测试，本次新开关遵循同一先例，未额外补充；真正的行为分支（是否显示卡片）
  已在后端用例中覆盖。

---

## 五、同步上游更新的操作流程

官方发布新版本时按本节执行。**方向永远是：上游 → `main` → `production`，绝不反向。**

**不需要每次官方有提交就同步。** 按自己的节奏，挑一个官方版本标签升级即可（见第 1 步）。

> **同步 `main` 不自动进行。** 只在仓库所有者明确要求时执行，或由他本人手动操作。
> AI 助手发现上游有新版本可以告知，但不得顺手合并，也不得在做其他任务时「顺便」同步。

功能分支不参与这条线，它走自己的「功能线」（见第六节）。两条线都汇入 `production`，
但要分成两次独立的合并，不要混在一起。

**标准节奏：先同步，再开发。** 手上有功能要做时，顺序是
①上游 → `main` ②`main` → `production` ③从 `production` 切功能分支 ④功能分支 → `production`。

下面的命令已用上游真实领先的 24 个提交完整实跑验证过。

### 0. 一次性配置（每台机器只做一次）

```bash
git remote add upstream https://github.com/QuantumNous/new-api.git

# 保险：确保 push 不会自动带上标签（默认就是 false，这里是防止全局配置被改过）
git config --local push.followTags false
```

用 `git config --get push.followTags` 可以随时确认，输出为空或 `false` 即安全。

### 1. 把官方更新同步进 `main`

**同步到版本标签，不要同步到 `upstream/main` 的最新提交。**
`upstream/main` 的 HEAD 是官方此刻正在开发的状态，可能包含没验证完的改动；
版本标签才是官方打包发版、构建 Docker 镜像的那个点。

先看有哪些版本可选：

```bash
git fetch upstream --tags
git tag -l 'v*' --sort=-v:refname | head        # 最新的若干个版本
git log --oneline <当前版本>..<目标版本>         # 这次会带进来哪些改动
```

官方的 Release 说明在 https://github.com/QuantumNous/new-api/releases ，
升级前先看一眼有没有破坏性改动、数据库迁移或配置变更。

确定目标版本后：

```bash
git checkout main
git merge --ff-only v1.0.0-rc.39                # 换成你选的标签
git push origin main                            # 只推分支，不要加 --tags
```

> 🚫 **绝对不要 `git push origin --tags` / `git push --follow-tags`。**
> 上游标签只拉到本地就够用了 —— 合并用的是本地标签，推不推到我们的仓库不影响任何事。
> 一旦把上游那几十个标签推上来，`docker-build.yml` 和 `release.yml` 会被逐个触发
> （它们的触发条件是 `tags: ['*']`，只排除了 `nightly*` / `*-alpha*`，`v1.0.0-rc.N` 照样命中），
> 一次性烧光 Actions 额度，还可能以你的名义发布镜像和 Release。详见本节末尾「关于 CI」。
>
> `git fetch upstream --tags` 只写本地，是安全的；`git push origin main` 默认也不带标签。

**必须用 `--ff-only`**。`main` 是官方的纯镜像，只应该快进。
如果这一步失败，说明有人往 `main` 提交过东西，`main` 已经不是纯镜像了 —— 先把那些提交挪到
功能分支上，不要用普通 merge 糊过去。

> **关于「稳定版」**：截至 2026-09-21，上游 1.x 只发布了 `v1.0.0-rc.N` 系列（当时最新 `rc.39`），
> **没有不带 `-rc` 后缀的正式版**；更早的是 `0.x.y` 系列。也就是说官方目前就是以 rc 对外发版的，
> 等「正式稳定版」可能要等很久。实际操作建议：选一个发布了几天、issue 区没有集中反馈问题的 rc，
> 不要追最新那个。
>
> 另外注意，我们的 `main` 当前停在 `v1.0.0-rc.38` 之后 8 个提交的位置（不在任何标签上）——
> 这是早期同步时直接跟了 `main` HEAD 造成的。下次同步到某个标签后就会重新对齐。
> 用 `git describe --tags main` 可以随时看 `main` 落在哪个版本附近。

### 2. 先在试验分支预演合并，不要直接动 `production`

```bash
git checkout -b sync-test production
git merge main
```

看冲突清单：

```bash
git diff --name-only --diff-filter=U
```

想在合并前就知道会不会冲突，可以不落盘预演：

```bash
git merge-tree --write-tree production main
```

### 3. 解冲突

对照「四、变更记录」里每条记录的**「修改的上游既有文件」**表 —— 那张表就是检查清单，
冲突只可能出现在表里列出的文件中。

- **语言包（`web/src/i18n/locales/*.json`）**：两边通常都只是新增文案，**保留两边的行**即可，
  删掉 `<<<<<<<` / `=======` / `>>>>>>>` 三行标记。改完务必确认 JSON 仍然合法。
- **其他文件**：我们的改动都是追加式的（注册一行、加一个 `case`、加一个分支），
  上游改的是别的地方，**同时保留两边**基本就是正确答案。
  如果发现上游把我们挂钩的位置整个重构掉了（比如 `GetTopUpInfo` 被改写、
  `model/option.go` 的配置注册方式换了），那就按上游的新写法把我们的钩子重新挂一遍，
  并在「四、变更记录」里补一条记录说明。

### 4. 验证（不能跳过）

> **本仓库的 GitHub Actions 目前是关闭的**，所以下面这些必须手动跑，没有自动把关。
> 详见本节末尾「关于 CI」。

```bash
# 后端
go build ./... && go vet ./controller/... ./service/... ./model/... ./setting/... ./router/...
go test ./controller/... ./model/... ./setting/...

# 前端
cd web && bun install && bun run typecheck && bun run test && bun run build
```

再针对我们自己加的功能做一次冒烟：后台能打开 PayerScan 配置分区、能下单拿到收款页、
回调能正常入账。

### 5. 合进 `production` 并发布

先留退路：

```bash
git branch production-backup-$(git rev-parse --short production) production
git push origin production-backup-$(git rev-parse --short production)
```

再正式合并：

```bash
git checkout production
git merge main          # 或者把验证通过的 sync-test 合进来
git push origin production
```

出问题就用 backup 分支回滚。

### 6. 补记录

在「四、变更记录」追加一条同步记录：同步到了上游哪个提交、冲突出现在哪些文件、怎么解的。
下次同步时这条记录就是参考。

### 关于 CI

CI（Continuous Integration，持续集成）指代码推上去之后，由 GitHub 自动在服务器上跑一遍
编译和测试，结果显示在 PR 页面上 —— 相当于一个不会忘事的门卫，替你挡住编译不过或测试挂掉的代码。

上游已经写好了配置，就在 `.github/workflows/ci.yml`，PR 一开就会跑：

- **Backend vet, build, and test** —— 后端 `go vet` / `go build` / `go test`
- **Frontend typecheck and test** —— 前端 `bun run typecheck` / `bun run test`

**但这个 fork 里它没有生效**：GitHub 对 fork 仓库默认关闭 Actions，需要手动开。
证据：PR #1 的 check runs 为 0，查 `ci.yml` 的运行记录返回 404（工作流未注册）。

**私有仓库还有必要开吗 —— 建议开。**

理由：这是个**处理真金白银的支付系统**，改错了不是页面难看，是钱对不上。
而人工验证最容易偷懒的恰恰是最慢的那一步：前端测试套件本地要跑 5～6 分钟，
赶时间时很容易跳过，CI 不会跳。私有仓库 Actions 有免费额度（GitHub Free 每月
2000 分钟，Pro 3000 分钟，具体以账号 Billing 页为准），按本项目一次 PR 约 10～15 分钟估算，
够跑一百多次 —— 对我们这种一个月改几次的节奏绰绰有余。

| | 开启 | 不开（现状） |
| --- | --- | --- |
| 把关 | 每次 PR 自动跑完整编译与测试 | 全靠本地手动跑，跳过了没人知道 |
| 成本 | 私有仓库计入免费额度，约 10～15 分钟/次 | 无 |
| 同步上游时 | 合完直接看结果 | 必须本地完整验证 |

开启方式：仓库 → Settings → Actions → General → 选 "Allow all actions"，
或在 Actions 标签页点 "I understand my workflows, go ahead and enable them"。

> ⚠️ **开启前必读：不要把上游的标签推到我们的仓库。**
> 上游另外三个工作流是**按标签触发**的，而且通配符很宽：
>
> | 工作流 | 触发条件 | 会不会被 `v1.0.0-rc.N` 触发 |
> | --- | --- | --- |
> | `docker-build.yml` | `push: tags: ['*', '!nightly*']` | **会** |
> | `release.yml` | `push: tags: ['*', '!*-alpha*']` | **会** |
> | `electron-build.yml` | `push: tags: ['*', '!*-*']` | 不会（排除了带 `-` 的预发布标签） |
>
> 也就是说，如果开了 Actions 又执行 `git push origin --tags`，把上游那几十个标签推上来，
> 会一次性触发几十轮 Docker 构建和发版流程，额度瞬间烧光，还可能以你的名义发布镜像和 Release。
>
> 规避：**同步时只 `git push origin main`，不要加 `--tags`**（默认行为就是不推标签，
> 别手动加）。`git fetch upstream --tags` 只拉到本地，是安全的。
> 如果不需要自己构建镜像，也可以直接在 Actions 页面把这三个工作流 Disable 掉，只留 `ci.yml`。

**在没开启之前，第 4 步的手动验证不能省。**

### 同步记录

| 日期 | 同步到的版本 | 冲突文件 | 处理方式 |
| --- | --- | --- | --- |
| 2026-09-21（预演，未合并） | `upstream/main` HEAD `9c293e8`（领先 24 个提交） | 7 个语言包，各 2 行 | 保留两边的文案行 |

> 上表这次是跟 `main` HEAD 做的预演。**后续一律同步到版本标签**，本列请填标签名（如 `v1.0.0-rc.39`）。

预演结果：Go 与前端代码 11 个文件全部自动合并成功；解完语言包冲突后
`go build` / `go vet` / `go test ./controller/... ./model/... ./setting/...` 全部通过。

---

## 六、功能分支的操作流程

我们自己改代码走这条线。和第五节的同步线分开走。

### 1. 先确认 `production` 是最新的

```bash
git fetch origin
git log --oneline origin/main..origin/production   # 应为空，否则先按第五节同步
```

`production` 落后于 `main` 就先同步，再开分支。不要在旧代码上开发。

### 2. 从 `production` 切分支

```bash
git checkout -b <feature-branch> origin/production
```

**不要从 `main` 切**，理由见第二节。

### 3. 开发

遵守第三节的核心约束：新增文件承载逻辑，对上游既有文件只做追加式最小改动。
改完在「四、变更记录」追加一条记录（这是强制的，见第一节）。

### 4. 提交前自检

```bash
# 只含我们自己的改动？（不应该出现官方提交）
git log --oneline origin/production..HEAD

# 实测能不能跟上游合并
git fetch upstream
git merge-tree --write-tree HEAD upstream/main
```

有冲突就调整改动位置，别留给以后。再跑第五节第 4 步的完整验证。

### 5. 开发期间上游更新了怎么办

先按第五节把 `production` 同步到最新，然后在功能分支上：

```bash
git merge production
```

**用 merge，不要 rebase** —— 分支已经推送且挂着 PR，rebase 会让别人的检出失效。

### 6. 合并发布

```bash
git checkout production
git merge <feature-branch>
git push origin production
```

合并前照例先打 backup 分支（见第五节第 5 步）。

---

## 七、记录模板

```markdown
### NNN — <改动标题>

| 项 | 内容 |
| --- | --- |
| 日期 | YYYY-MM-DD |
| 分支 | <branch> |
| PR | <url> |
| 提交 | <sha> |
| 基线 | <上游 sha> |

#### 需求
<为什么要改，需求原话或链接>

#### 新增文件
| 文件 | 作用 |
| --- | --- |

#### 修改的上游既有文件
| 文件 | 改了什么 |
| --- | --- |

#### 新增路由 / 配置项
<如无可省略>

#### 设计要点
<关键取舍，尤其是安全、并发、金额相关的部分>

#### 验证结果
| 检查 | 结果 |
| --- | --- |

#### 已知限制 / 待办
<如无写「无」>
```
