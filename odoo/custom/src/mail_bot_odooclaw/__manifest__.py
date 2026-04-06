{
    "name": "OdooClaw AI Bot",
    "version": "19.0.1.0.0",
    "category": "Discuss",
    "summary": "Integrate OdooClaw AI agent via webhooks in Odoo Discuss",
    "author": "Nicolás Ramos",
    "license": "AGPL-3",
    "depends": ["mail"],
    "data": [
        "security/ir.model.access.csv",
        "data/odooclaw_bot_data.xml",
        "data/odooclaw_report_cron.xml",
        "views/odooclaw_report_views.xml",
    ],
    "installable": True,
    "application": False,
    "auto_install": False,
    "maintainer": "nicolasramos",
}
