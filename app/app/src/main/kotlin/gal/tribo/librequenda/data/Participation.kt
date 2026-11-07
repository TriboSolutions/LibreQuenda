package gal.tribo.librequenda.data

import org.json.JSONObject
import java.time.Instant
import java.time.OffsetDateTime

/** One of the council's announcements, on its pages until it expires. */
data class CouncilNotice(val id: String, val title: String, val body: String, val createdAt: Instant, val expiresAt: Instant)

internal fun JSONObject.toCouncilNotice() = CouncilNotice(
    id = getString("id"),
    title = optString("title"),
    body = optString("body"),
    createdAt = OffsetDateTime.parse(getString("created_at")).toInstant(),
    expiresAt = OffsetDateTime.parse(getString("expires_at")).toInstant(),
)

/** A problem reported and not fixed yet, by category: never what anyone wrote, nor who. */
data class KnownIssue(val category: String, val reports: Int, val firstReportedAt: Instant, val inProgress: Boolean)

data class SolvedIssue(val category: String, val solvedAt: Instant)

/** What is known to be broken at a facility, what was fixed lately, and how long the council takes on average. */
data class IssuesSummary(val open: List<KnownIssue>, val recentlySolved: List<SolvedIssue>, val averageDays: Double?, val solvedCount: Int)

internal fun JSONObject.toIssuesSummary(): IssuesSummary {
    val open = optJSONArray("open")
    val solved = optJSONArray("recently_solved")
    return IssuesSummary(
        open = List(open?.length() ?: 0) { open!!.getJSONObject(it) }.map {
            KnownIssue(it.optString("category"), it.optInt("reports"), OffsetDateTime.parse(it.getString("first_reported_at")).toInstant(), it.optBoolean("in_progress"))
        },
        recentlySolved = List(solved?.length() ?: 0) { solved!!.getJSONObject(it) }.map {
            SolvedIssue(it.optString("category"), OffsetDateTime.parse(it.getString("solved_at")).toInstant())
        },
        averageDays = if (isNull("average_days_to_solve")) null else optDouble("average_days_to_solve"),
        solvedCount = optInt("solved_count"),
    )
}

/** An answer; votes is null while the poll is open. */
data class PollOption(val id: String, val label: String, val votes: Int?)

/** A council's poll, as a resident sees it: one vote, which can change while it is open. */
data class Poll(
    val id: String,
    val question: String,
    val councilName: String,
    val facilityName: String,
    val residentsOnly: Boolean,
    val open: Boolean,
    val closesAt: Instant, // or when it closed, if early
    val myOption: String, // "" before voting
    val votes: Int?, // null while open
    val options: List<PollOption>,
)

/** One page of the polls, and whether there are more after it. */
data class PollsPage(val polls: List<Poll>, val hasMore: Boolean)

/** GET /me/polls?page=: the polls of the councils the person is part of, the open ones first, 20 a page. */
suspend fun fetchPolls(token: String, page: Int = 1): PollsPage {
    val json = Api.getJson(Api.baseUrl + "/me/polls" + (if (page > 1) "?page=$page" else ""), token)
    val polls = json.optJSONArray("polls")
    val list = List(polls?.length() ?: 0) { polls!!.getJSONObject(it) }.map { p ->
        val options = p.optJSONArray("options")
        Poll(
            id = p.getString("id"),
            question = p.optString("question"),
            councilName = p.optString("tenant_name"),
            facilityName = p.optString("facility_name"),
            residentsOnly = p.optBoolean("residents_only"),
            open = p.optBoolean("open"),
            closesAt = OffsetDateTime.parse(p.getString("closes_at")).toInstant(),
            myOption = p.optString("my_option"),
            votes = if (p.isNull("votes")) null else p.optInt("votes"),
            options = List(options?.length() ?: 0) { options!!.getJSONObject(it) }.map {
                PollOption(it.getString("id"), it.optString("label"), if (it.isNull("votes")) null else it.optInt("votes"))
            },
        )
    }
    return PollsPage(list, json.optJSONObject("pagination")?.optBoolean("has_more") == true)
}

/** PUT /me/polls/{id}/vote: casts, or changes, the person's vote. */
suspend fun vote(token: String, pollId: String, optionId: String) {
    Api.send("PUT", Api.baseUrl + "/me/polls/$pollId/vote", token, JSONObject().put("option_id", optionId))
}
