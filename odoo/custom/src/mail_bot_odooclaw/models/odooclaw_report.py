import uuid
from datetime import timedelta
from odoo import api, fields, models


class OdooclawReport(models.Model):
    _name = "odooclaw.report"
    _description = "OdooClaw Visual Report"
    _order = "create_date desc"

    name = fields.Char(string="Title", required=True)
    token = fields.Char(
        string="Access Token",
        required=True,
        index=True,
        default=lambda self: str(uuid.uuid4()),
        copy=False,
    )
    html_content = fields.Html(string="HTML Content", required=True, sanitize=False)
    res_id = fields.Integer(string="Related Record ID")
    res_model = fields.Char(string="Related Model")
    create_date = fields.Datetime(string="Created On", readonly=True)

    _sql_constraints = [
        ("token_unique", "UNIQUE(token)", "Report token must be unique."),
    ]

    @api.model
    def _cron_cleanup_expired(self):
        expiry_date = fields.Datetime.now() - timedelta(days=7)
        expired = self.search([("create_date", "<", expiry_date)])
        if expired:
            expired.unlink()
