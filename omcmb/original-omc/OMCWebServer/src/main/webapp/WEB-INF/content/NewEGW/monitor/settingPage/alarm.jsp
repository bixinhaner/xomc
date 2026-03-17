<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style type="text/css">
#egwAlarmPage{
    width:100%;
    height:calc(100% - 10px);
}
#egwAlarmPage .itemMainBoxCls{
	border:1px solid #d5dcec;
	border-radius:10px;
	background:#fff;
	padding:0px 20px;
	height:100%;
    display: flex;
    flex-direction: column;
    position: relative;
}
#egwAlarmPage .itemMainBoxTitle {
	height:50px;
    line-height: 50px;
	font-size:14px;
	font-weight:bold;
}
#egwAlarmPage .alarmItemTableBoxCls {
	flex: 1;
}
#egwAlarmPage .alarmTableBoxCls{
    height:calc(100% - 60px);
    border:1px solid #d5dcec;
    box-sizing: border-box;
}
.egwAlarmMinor,.egwAlarmMajor,.egwAlarmCritical,.egwAlarmWarning{
    display: flex;
    align-items: center;
}
.egwAlarmMinor .el-icon:before{
    color: #FFDA41;
    font-size: 20px;
}
.egwAlarmMajor .el-icon:before{
    color: #FF973E;
    font-size: 20px;
}
.egwAlarmCritical .el-icon:before{
    color: #FC5959;
    font-size: 20px;
}
.egwAlarmWarning .el-icon:before{
    color: #60BEFC;
    font-size: 20px;
}
.egwUnconfirmInactive,.egwConfirmInactive,.egwUnconfirmActive,.egwConfirmActive{
    display: flex;
    align-items: center;
}
.egwUnconfirmInactive .el-icon:before{
    color: #E88282;
    font-size: 20px;
}
.egwConfirmInactive .el-icon:before{
    color: #E88282;
    font-size: 20px;
}
.egwUnconfirmActive .el-icon:before{
    color: #67D972;
    font-size: 20px;
}
.egwConfirmActive .el-icon:before{
    color: #67D972;
    font-size: 20px;
}
.AlarmInfoDialogCls .el-dialog__body{
    height: 610px;
    padding: 0px  20px 20px 20px;
}
#alarmThresholdDialog .alarmThresholdItem .el-input__inner{
    width:110px !important;
}
#alarmThresholdDialog .alarmThresholdItem .el-input-group__append{
    padding-left: 50px!important;
}
#alarmThresholdDialog  .alarmThresholdBoxCls{
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    margin: 0px 0px 20px 0px;
}
#alarmThresholdDialog  .alarmThresholdBoxCls .el-form-item{
    margin-bottom: 0px !important;
}
#alarmThresholdDialog  .alarmThresholdBoxCls .el-form-item__error{
    top:25px;
    white-space: nowrap;
}
#alarmThresholdDialog  .alarmThresholdBoxCls .el-input-group__append{
    padding: 0px 10px!important;
}
#alarmThresholdDialog .alarmThresholdHeadCls{
    width: 360px;
}
#alarmThresholdDialog .alarmThresholdItemLabelCls{
    padding: 0px 10px 0px 30px;
}
#alarmThresholdDialog .alarmThresholdItemRightLabelCls{
    margin-left: 100px;
}
#alarmThresholdDialog  .promptCls{
    font-size: 12px;
    color: #BBBBBB;
    position: relative;
    top: 0px;
    margin-bottom: 20px;
}
#alarmThresholdDialog  .promptCls .el-icon:before{
    color: #BBBBBB;
    font-size: 12px;
}
#alarmThresholdDialog .el-dialog__body{
    padding: 20px 30px 15px;
}
#alarmThresholdDialog .el-dialog__headerbtn:hover .el-dialog__close{
    color:var(--main-color);
}
#egwAlarmDetailPages .alarmMinorIcon::before,
#egwAlarmPage .alarmMinorIcon::before{
	color: #FFDA41;
	font-size: 18px;
}
#egwAlarmDetailPages .alarmMajorIcon::before,
#egwAlarmPage .alarmMajorIcon::before{
	color: #FF973E;
	font-size: 18px;
}
#egwAlarmDetailPages .alarmCriticalIcon::before,
#egwAlarmPage .alarmCriticalIcon::before{
	color: #FC5959;
	font-size: 18px;
}
#egwAlarmDetailPages .alarmWarningIcon::before,
#egwAlarmPage .alarmWarningIcon::before{
	color: #60BEFC;
	font-size: 18px;
}
#egwAlarmDetailPages label{
	color: #7A7992;
	margin-right:5px;
	text-align:left;
	display:inline-block;
	width:200px;
}
#egwAlarmDetailPages span{
	color: rgba(0, 0, 0, 0.8);
	width: 730px;
	word-wrap: break-word;
}
#egwAlarmDetailPages div{
	margin-bottom:20px;
	font-size:12px;
	display: flex;
	width: 940px;
}
</style>
<div id="egwAlarmPage">
	<div class="itemMainBoxCls">
        <div v-show="optBtnShow" class="newIconBoxCls-bt" style="right:10px;top:10px;" tip="<%=rb.getString("SheZhi")%>" @click="egwAlarmSetting">		
            <span class="el-icon-operation-settings el-icon"></span>
        </div>
        <div class="alarmItemTableBoxCls">
            <div class="itemMainBoxTitle"><%=rb.getString("HuoDongGaoJing")%></div>
            <div class="alarmTableBoxCls">
				<el-ctable ref="activeAlarmTable" :rownumber="true" :time="6" id="activeAlarmTable" :url="activeAlarmTableUrl" :query-params="activeQueryParams" height="100%" pagination="true">
                    <el-table-column label='' width="40" prop="">
                        <template slot-scope="scope">
                            <div class="el-icon el-icon-operation-more-circle" @click="optActiveAlarmClick(scope.row,event)" v-clickoutside="handerClose" style="cursor: pointer;"></div>
                        </template>
                    </el-table-column>
                    <el-table-column label='<%=rb.getString("GaoJingJiBie")%>' min-width="100"  prop="alarm_serverity_value" sortable>
                        <template slot-scope="scope">
                            <div v-if="scope.row.alarm_serverity_value == 'Minor'" class="egwAlarmMinor">
                                <span class="el-icon el-icon-status-alarm" style="margin-right:5px;"></span><%=rb.getString("CiYaoGaoJing")%>
                            </div>
                            <div v-else-if="scope.row.alarm_serverity_value == 'Major'"  class="egwAlarmMajor">
                                <span class="el-icon el-icon-status-alarm" style="margin-right:5px;"></span><%=rb.getString("ZhuYaoGaoJing")%>
                            </div>
                            <div v-else-if="scope.row.alarm_serverity_value == 'Critical'" class="egwAlarmCritical">
                                <span class="el-icon el-icon-status-alarm" style="margin-right:5px;"></span><%=rb.getString("JinJiGaoJing")%>
                            </div>
                            <div v-else-if="scope.row.alarm_serverity_value == 'Warning'" class="egwAlarmWarning">
                                <span class="el-icon el-icon-status-alarm" style="margin-right:5px;"></span><%=rb.getString("JingGaoGaoJing")%>
                            </div>
                        </template>
                    </el-table-column>
                    <el-table-column label='<%=rb.getString("GaoJingWeiYiBiaoZhi")%>' min-width="130" prop="alarm_identifier"></el-table-column>
                    <el-table-column label='<%=rb.getString("KeNengYuanYin")%>' min-width="180" prop="alarm_name" show-overflow-tooltip></el-table-column>
                    <el-table-column label='<%=rb.getString("GaoJingZhuangTai")%>' min-width="190" prop="deal_state"  show-overflow-tooltip sortable>
                        <template slot-scope="scope">
                            <div v-if="scope.row.deal_state == '0'" class="egwUnconfirmInactive">
                                <span class="el-icon el-icon-status-unconfirmActive" style="margin-right:5px;"></span><%=rb.getString("WeiQueRenWeiQingChu")%>
                            </div>
                            <div v-else-if="scope.row.deal_state == '1'" class="egwConfirmInactive">
                                <span class="el-icon el-icon-status-confirmActive" style="margin-right:5px;"></span><%=rb.getString("YiQueRenWeiQingChu")%>
                            </div>
                            <div v-if="scope.row.deal_state == '2'" class="egwUnconfirmActive">
                                <span class="el-icon el-icon-status-unconfirmActive" style="margin-right:5px;"></span><%=rb.getString("WeiQueRenYiQingChu")%>
                            </div>
                            <div v-else-if="scope.row.deal_state == '3'" class="egwConfirmActive">
                                <span class="el-icon el-icon-status-confirmActive" style="margin-right:5px;"></span><%=rb.getString("YiQueRenYiQingChu")%>
                            </div>
                        </template>
                    </el-table-column>
                    <el-table-column label='<%=rb.getString("GuZhangShiJian")%>' min-width="150" prop="event_time" sortable></el-table-column>
                    <el-table-column label='<%=rb.getString("GengXinShiJian")%>' width="150"  prop="upd_time" sortable></el-table-column>
                    <el-table-column label='<%=rb.getString("GaoJingCiShu")%>'  min-width="110" prop="alarm_count" sortable></el-table-column>
                </el-ctable>
                <el-cmenu ref="activeAlarmMenus" :data="activeAlarmMenusData" @click="clickAlarmMenu"></el-cmenu>
            </div>
        </div>
		<div class="alarmItemTableBoxCls">
            <div class="itemMainBoxTitle"><%=rb.getString("LiShiGaoJing")%></div>
             <div class="alarmTableBoxCls">
				<el-ctable ref="historyAlarmTable" :rownumber="true" :time="6" id="historyAlarmTable" :url="historyAlarmTableUrl" :query-params="historyQueryParams" height="100%" pagination="true">
                    <el-table-column label='' width="40" prop="">
                        <template slot-scope="scope">
                            <div class="el-icon el-icon-operation-more-circle" @click="optHistoryAlarmClick(scope.row,event)" v-clickoutside="handerClose" style="cursor: pointer;"></div>
                        </template>
                    </el-table-column>
                    <el-table-column label='<%=rb.getString("GaoJingJiBie")%>' min-width="100"  prop="alarm_serverity_value" sortable>
                        <template slot-scope="scope">
                            <div v-if="scope.row.alarm_serverity_value == 'Minor'" class="egwAlarmMinor">
                                <span class="el-icon el-icon-status-alarm" style="margin-right:5px;"></span><%=rb.getString("CiYaoGaoJing")%>
                            </div>
                            <div v-else-if="scope.row.alarm_serverity_value == 'Major'"  class="egwAlarmMajor">
                                <span class="el-icon el-icon-status-alarm" style="margin-right:5px;"></span><%=rb.getString("ZhuYaoGaoJing")%>
                            </div>
                            <div v-else-if="scope.row.alarm_serverity_value == 'Critical'" class="egwAlarmCritical">
                                <span class="el-icon el-icon-status-alarm" style="margin-right:5px;"></span><%=rb.getString("JinJiGaoJing")%>
                            </div>
                            <div v-else-if="scope.row.alarm_serverity_value == 'Warning'" class="egwAlarmWarning">
                                <span class="el-icon el-icon-status-alarm" style="margin-right:5px;"></span><%=rb.getString("JingGaoGaoJing")%>
                            </div>
                        </template>
                    </el-table-column>
                    <el-table-column label='<%=rb.getString("GaoJingWeiYiBiaoZhi")%>' min-width="130" prop="alarm_identifier"></el-table-column>
                    <el-table-column label='<%=rb.getString("KeNengYuanYin")%>' min-width="180" prop="alarm_name" show-overflow-tooltip></el-table-column>
                    <el-table-column label='<%=rb.getString("GaoJingZhuangTai")%>' min-width="190" prop="deal_state"  show-overflow-tooltip sortable>
                        <template slot-scope="scope">
                            <div v-if="scope.row.deal_state == '0'" class="egwUnconfirmInactive">
                                <span class="el-icon el-icon-status-unconfirmActive" style="margin-right:5px;"></span><%=rb.getString("WeiQueRenWeiQingChu")%>
                            </div>
                            <div v-else-if="scope.row.deal_state == '1'" class="egwConfirmInactive">
                                <span class="el-icon el-icon-status-confirmActive" style="margin-right:5px;"></span><%=rb.getString("YiQueRenWeiQingChu")%>
                            </div>
                            <div v-if="scope.row.deal_state == '2'" class="egwUnconfirmActive">
                                <span class="el-icon el-icon-status-unconfirmActive" style="margin-right:5px;"></span><%=rb.getString("WeiQueRenYiQingChu")%>
                            </div>
                            <div v-else-if="scope.row.deal_state == '3'" class="egwConfirmActive">
                                <span class="el-icon el-icon-status-confirmActive" style="margin-right:5px;"></span><%=rb.getString("YiQueRenYiQingChu")%>
                            </div>
                        </template>
                    </el-table-column>
                    <el-table-column label='<%=rb.getString("GuZhangShiJian")%>' min-width="150" prop="event_time" sortable></el-table-column>
                    <el-table-column label='<%=rb.getString("GaoJingQingChuShiJian")%>' width="150"  prop="clear_time" sortable></el-table-column>
                    <el-table-column label='<%=rb.getString("GaoJingCiShu")%>'  min-width="110" prop="alarm_count" sortable></el-table-column>
                </el-ctable>
                <el-cmenu ref="historyAlarmMenus" :data="historyAlarmMenusData" @click="clickAlarmMenu"></el-cmenu>
            </div>
        </div>
	</div>
    <!--告警详情弹窗-->
	<el-dialog :title="'<%=rb.getString("XiangXiXinXi")%>'" :visible.sync="showAlarmInfo" :close-on-click-modal="false" @close="alarmInfoDialogClose" :append-to-body="true" top="15vh" width="1000" >
        <div id='egwAlarmDetailPages'>
            <div>
                <label><%=rb.getString("XuHao")%></label>
                <span>{{ALARM_ID}}</span>
            </div>
            <div>
                <label><%=rb.getString("GaoJingWeiYiBiaoZhi")%></label>
                <span>{{ALARM_IDENTIFIER}}</span>
            </div>
            <div>
                <label><%=rb.getString("KeNengYuanYin")%></label>
                <span>{{ALARM_NAME}}</span>
            </div>
            <div>
                <label><%=rb.getString("JuTiGuZhang")%></label>
                <span>{{SPECIFIC_PROBLEM}}</span>
            </div>
            <div>
                <label><%=rb.getString("FuJianXinXi")%></label>
                <span>{{ADDITIONAL_INFORMATION}}</span>
            </div>
            <div>
                <label><%=rb.getString("FuJianWenBen")%></label>
                <span>{{ADDITIONAL_TEXT}}</span>
            </div>
            <div class='commonFlex'>
                <label><%=rb.getString("GaoJingJiBie")%></label>
                <div v-if="ALARM_SERVERITY == 'Minor'" style='margin-bottom: 0;width: 730px;'>
                    <span class="el-icon el-icon-status-alarm alarmMinorIcon" style="margin-right:5px;width: auto;"></span><%=rb.getString("CiYaoGaoJing")%>
                </div>
                <div v-else-if="ALARM_SERVERITY == 'Major'" style='margin-bottom: 0;width: 730px;'>
                    <span class="el-icon el-icon-status-alarm alarmMajorIcon" style="margin-right:5px;width: auto;"></span><%=rb.getString("ZhuYaoGaoJing")%>
                </div>
                <div v-else-if="ALARM_SERVERITY == 'Critical'" style='margin-bottom: 0;width: 730px;'>
                    <span class="el-icon el-icon-status-alarm alarmCriticalIcon" style="margin-right:5px;width: auto;"></span><%=rb.getString("JinJiGaoJing")%>
                </div>
                <div v-else-if="ALARM_SERVERITY == 'Warning'" style='margin-bottom: 0;width: 730px;'>
                    <span class="el-icon el-icon-status-alarm alarmWarningIcon" style="margin-right:5px;width: auto;"></span><%=rb.getString("JingGaoGaoJing")%>
                </div>
            </div>
            <div>
                <label><%=rb.getString("ShiJianLeiXing")%></label>
                <span>{{EVENT_TYPE}}</span>
            </div>
            <div>
                <label><%=rb.getString("XinGaoJingYuan")%></label>
                <span>{{NE_TYPE}}</span>
            </div>
            <div>
                <label><%=rb.getString("WangYuanDingWei")%></label>
                <span>{{EQUIP_INFO}}</span>
            </div>
           <div class='commonFlex'>
                <label><%=rb.getString("GaoJingZhuangTai")%></label>
                <span v-if='false'>{{DEAL_STATE}}</span>
                <div v-if="DEAL_STATE == '0'" style="display:flex;align-items: center; margin-bottom: 0; width: 730px;">
                    <span class="el-icon el-icon-status-unconfirmActive redIcon" style="margin-right:5px;font-size: 22px; width: auto;"></span><%=rb.getString("WeiQueRenWeiQingChu")%>
                </div>
                <div v-else-if="DEAL_STATE == '1'" style="display:flex;align-items: center;margin-bottom: 0;width: 730px;">
                    <span class="el-icon el-icon-status-confirmActive redIcon" style="margin-right:5px;font-size: 22px;width: auto;"></span><%=rb.getString("YiQueRenWeiQingChu")%>
                </div>
                <div v-if="DEAL_STATE == '2'" style="display:flex;align-items: center;margin-bottom: 0;width: 730px;">
                    <span class="el-icon el-icon-status-unconfirmActive greenIcon" style="margin-right:5px;font-size: 22px;width: auto;"></span><%=rb.getString("WeiQueRenYiQingChu")%>
                </div>
                <div v-else-if="DEAL_STATE == '3'" style="display:flex;align-items: center;margin-bottom: 0;width: 730px;">
                    <span class="el-icon el-icon-status-confirmActive greenIcon" style="margin-right:5px;font-size: 22px;width: auto;"></span><%=rb.getString("YiQueRenYiQingChu")%>
                </div>
            </div>
            <div>
                <label><%=rb.getString("GuZhangShiJian")%></label>
                <span>{{EVENT_TIME}}</span>
            </div>
            <div>
                <label><%=rb.getString("GengXinShiJian")%></label>
                <span>{{UPD_TIME}}</span>
            </div>
            <div v-if="confirmTipFlag">
                <label><%=rb.getString("QueRenRen")%></label>
                <span>{{DEAL_USER}}</span>
            </div>
            <div v-if="confirmTipFlag">
                <label><%=rb.getString("QueRenShiJian")%></label>
                <span>{{DEAL_TIME}}</span>
            </div>
            <div v-if="clearFlag">
                <label><%=rb.getString("GaoJingQingChuRen")%></label>
                <span>{{CLEAR_USER}}</span>
            </div>
            <div v-if="clearFlag">
                <label><%=rb.getString("GaoJingQingChuShiJian")%></label>
                <span>{{ClEAR_TIME}}</span>
            </div>
            <div style="display: flex;">
                <label style="min-width: 165px"><%=rb.getString("ChuLiJianYi")%></label>
                <span>{{SUGGESTION}}</span>
            </div>
            <div>
                <label><%=rb.getString("MiaoShu")%></label>
                <span>{{DEAL_MEMO}}</span>
            </div>
        </div>
    </el-dialog>
	<!--告警清除弹窗-->
	<el-dialog :title="'<%=rb.getString("QueRen")%>'" :visible.sync="showClearAlarmDialog" width="500" 
		:close-on-click-modal="false" top="15vh" @close="clearAlarmDialogClose" :append-to-body="true">
		<el-form  :model="clearAlarmForm" ref="clearAlarmForm" label-position="top">
			<el-form-item>
				<%=rb.getString("QueRenQingChuGaoJing")%>
			</el-form-item>
			<el-form-item label="<%=rb.getString("MiaoShu")%>" prop="description">
				<el-input type="textarea" :rows="3" v-model="clearAlarmForm.description">
			</el-form-item>
		</el-form>
		<span slot="footer" class="dialog-footer">
            <el-button type="primary" @click="clearAlarmSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="clearAlarmDialogClose"><%=rb.getString("QuXiao")%></el-button>
		</span>
	</el-dialog>
	<!--告警确认弹窗-->
	<el-dialog :title="'<%=rb.getString("QueRenGaoJing")%>'" :visible.sync="showConfirmAlarmDialog" width="500" 
		:close-on-click-modal="false" top="15vh" @close="confirmAlarmDialogClose" :append-to-body="true">
		<el-form :model="alarmConfirmForm" ref="alarmConfirmForm" label-position="top">
			<el-form-item v-if="alarmConfirmFlag" label="<%=rb.getString("QueRenRen")%>" prop="confirmUser">
				<el-input :disabled="true" v-model="alarmConfirmForm.confirmUser"></el-input>
			</el-form-item>
			<el-form-item v-if="alarmConfirmFlag" label="<%=rb.getString("QueRenShiJian")%>" prop="confirmTime">
				<el-input :disabled="true" v-model="alarmConfirmForm.confirmTime">
			</el-form-item>
			<el-form-item label="<%=rb.getString("MiaoShu")%>" prop="description">
				<el-input type="textarea" :rows="3" maxlength="500" v-model="alarmConfirmForm.description">
			</el-form-item>
		</el-form>
		<span slot="footer" class="dialog-footer">
            <el-button type="primary" @click="confirmAlarmSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="confirmAlarmDialogClose"><%=rb.getString("QuXiao")%></el-button>
		</span>
	</el-dialog>
    <!--告警阙值配置弹窗-->
	<el-dialog id="alarmThresholdDialog" :title="'<%=rb.getString("GaoJingYuZhiPeiZhi")%>'" :visible.sync="showAlarmThresholdDialog" width="60%" 
		:close-on-click-modal="false" top="15vh" @close="alarmThresholdDialogClose" append-to-body>
        <div :class="loadingDialogShow ? 'loading' : ''" style="position:relative;padding-bottom:20px;">
            <el-form :model="alarmThresholdForm" :rules="alarmThresholdRules" ref="alarmThresholdForm" label-position="top">
                <div class="promptCls"><span class="el-icon-circle-info el-icon" style="margin-right:5px;" ></span><%=rb.getString("GaoJingYuZhiPeiZhiTiShi")%></div>
                <div class="alarmThresholdBoxCls">
                    <div class="alarmThresholdHeadCls">(<%=rb.getString("AlarmId")%>:31002)<%=rb.getString("CPUShiYongLv")%></div>
                    <div class="alarmThresholdItemLabelCls"><%=rb.getString("GaoJingZhi")%></div>
                    <el-form-item prop="cpuUseRate" label=""  label-width="160px">
                        <el-input style='width:100px;' v-model="alarmThresholdForm.cpuUseRate" @change="commonConfigValid('cpuUseClearRate')">
                            <template slot="append">%</template>
                        </el-input>
                    </el-form-item>
                    <div class="alarmThresholdItemLabelCls alarmThresholdItemRightLabelCls"><%=rb.getString("HuiFuZhi")%></div>
                    <el-form-item prop="cpuUseClearRate" label=""  label-width="160px">
                        <el-input style='width:100px;' v-model="alarmThresholdForm.cpuUseClearRate" @change="commonConfigValid('cpuUseRate')">
                            <template slot="append">%</template>
                        </el-input>
                    </el-form-item>
                </div>
                <div class="alarmThresholdBoxCls">
                    <div class="alarmThresholdHeadCls">(<%=rb.getString("AlarmId")%>:31003)<%=rb.getString("NeiCunShiYongLv")%></div>
                    <div class="alarmThresholdItemLabelCls"><%=rb.getString("GaoJingZhi")%></div>
                    <el-form-item prop="memoryUseRate" label=""  label-width="160px">
                        <el-input style='width:100px;' v-model="alarmThresholdForm.memoryUseRate" @change="commonConfigValid('memoryClearUseRate')">
                            <template slot="append">%</template>
                        </el-input>
                    </el-form-item>
                    <div class="alarmThresholdItemLabelCls alarmThresholdItemRightLabelCls"><%=rb.getString("HuiFuZhi")%></div>
                    <el-form-item prop="memoryClearUseRate" label=""  label-width="160px">
                        <el-input style='width:100px;' v-model="alarmThresholdForm.memoryClearUseRate" @change="commonConfigValid('memoryUseRate')">
                            <template slot="append">%</template>
                        </el-input>
                    </el-form-item>
                </div>
                <div class="alarmThresholdBoxCls">
                    <div class="alarmThresholdHeadCls">(<%=rb.getString("AlarmId")%>:31004)<%=rb.getString("CiPanShiYongLv")%></div>
                    <div class="alarmThresholdItemLabelCls"><%=rb.getString("GaoJingZhi")%></div>
                    <el-form-item prop="diskUseRate" label=""  label-width="160px">
                        <el-input style='width:100px;' v-model="alarmThresholdForm.diskUseRate" @change="commonConfigValid('diskClearUseRate')">
                            <template slot="append">%</template>
                        </el-input>
                    </el-form-item>
                    <div class="alarmThresholdItemLabelCls alarmThresholdItemRightLabelCls"><%=rb.getString("HuiFuZhi")%></div>
                    <el-form-item prop="diskClearUseRate" label=""  label-width="160px">
                        <el-input style='width:100px;' v-model="alarmThresholdForm.diskClearUseRate" @change="commonConfigValid('diskUseRate')">
                            <template slot="append">%</template>
                        </el-input>
                    </el-form-item>
                </div>
                <div class="alarmThresholdBoxCls">
                    <div class="alarmThresholdHeadCls">(<%=rb.getString("AlarmId")%>:33003)<%=rb.getString("IPsecSuiDaoShuChaoGuoXianZhi")%></div>
                    <div class="alarmThresholdItemLabelCls"><%=rb.getString("GaoJingZhi")%></div>
                    <el-form-item prop="ikeSAMoreLicenseRate" label=""  label-width="160px">
                        <el-input style='width:100px;' v-model="alarmThresholdForm.ikeSAMoreLicenseRate" @change="commonConfigValid('ikeSAMoreLicenseClearRate')">
                            <template slot="append">%</template>
                        </el-input>
                    </el-form-item>
                    <div class="alarmThresholdItemLabelCls alarmThresholdItemRightLabelCls"><%=rb.getString("HuiFuZhi")%></div>
                    <el-form-item prop="ikeSAMoreLicenseClearRate" label=""  label-width="160px">
                        <el-input style='width:100px;' v-model="alarmThresholdForm.ikeSAMoreLicenseClearRate" @change="commonConfigValid('ikeSAMoreLicenseRate')">
                            <template slot="append">%</template>
                        </el-input>
                    </el-form-item>
                </div>
                <div class="alarmThresholdBoxCls">
                    <div class="alarmThresholdHeadCls">(<%=rb.getString("AlarmId")%>:33001)<%=rb.getString("eNBJieRuLv")%></div>
                    <div class="alarmThresholdItemLabelCls"><%=rb.getString("GaoJingZhi")%></div>
                    <el-form-item prop="enbMoreLicenseRate" label=""  label-width="160px">
                        <el-input style='width:100px;' v-model="alarmThresholdForm.enbMoreLicenseRate" @change="commonConfigValid('enbMoreLicenseClearRate')">
                            <template slot="append">%</template>
                        </el-input>
                    </el-form-item>
                    <div class="alarmThresholdItemLabelCls alarmThresholdItemRightLabelCls"><%=rb.getString("HuiFuZhi")%></div>
                    <el-form-item prop="enbMoreLicenseClearRate" label=""  label-width="160px">
                        <el-input style='width:100px;' v-model="alarmThresholdForm.enbMoreLicenseClearRate" @change="commonConfigValid('enbMoreLicenseRate')">
                            <template slot="append">%</template>
                        </el-input>
                    </el-form-item>
                </div>
                
            </el-form>
        </div>
		<span slot="footer" class="dialog-footer" v-show="!loadingDialogShow">
            <el-button type="primary" @click="alarmThresholdSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="showAlarmThresholdDialog = false"><%=rb.getString("QuXiao")%></el-button>
		</span>
	</el-dialog>
</div>
<script>
var egwAlarmPage = new Vue({
	el: '#egwAlarmPage', 
	data() {
		var vm = this;
        var validateAlarmValOrRegainVal = function(rule,value,callback) {
            var contrast = rule.contrast;
            var type = rule.type;
            if( value === '' || value === null || value === undefined) {
                callback('<%=rb.getString("FanWei")%>1~100,Integer');
            }else {
                if(vm.isNumeric(value)&&parseInt(value)>=1 && parseInt(value)<=100){
                    if(vm.alarmThresholdForm[contrast]){
                        if(type == 'max'){
                            if(parseInt(value) <= parseInt(vm.alarmThresholdForm[contrast])){
                                callback('<%=rb.getString("GaoJingZhiYingDaYuHuiFuZhiTiShi")%>');
                            }else{
                                callback();
                            }
                        }else{
                            if(parseInt(value) >= parseInt(vm.alarmThresholdForm[contrast])){
                                callback('<%=rb.getString("HuiFuZhiYingXiaoYuGaoJingZhiTiShi")%>');
                            }else{
                                callback();
                            }
                        }
                    }else{
                        callback();
                    }
                }else{
                    callback('<%=rb.getString("FanWei")%>1~100,Integer')
                }
            }
        };
		return {
            egwCode:'',
            rowDataAlarm:'',
            activeAlarmTableUrl:'',
            historyAlarmTableUrl:'',
            activeQueryParams:{
                timeZone:timeZone,
                alarmType:'ACTIVE',
                queryType:'View',
                alarmServerity:'31001,31002,31003,31004',
                deviceCode:'',
                neType:'EGW'
            },
            historyQueryParams:{
                timeZone:timeZone,
                alarmType:'HISTORY',
                queryType:'View',
                alarmServerity:'31001,31002,31003,31004',
                deviceCode:'',
                neType:'EGW'
            },
            activeAlarmMenusData:[],
            historyAlarmMenusData:[],
            showClearAlarmDialog:false,
            clearAlarmForm:{
                description:'',
            },
            showConfirmAlarmDialog:false,
            alarmConfirmFlag:false,
            alarmConfirmForm:{
                confirmUser:'',
                confirmTime:'',
                description:'',
            },
			alarmInfoDialogUrl:'',
            showAlarmInfo:false,
            alarmInfoDialogWidth:'850px',
            showAlarmThresholdDialog:false,
            alarmThresholdForm:{
                cpuUseRate:'',
                cpuUseClearRate:'',
                memoryUseRate:'',
                memoryClearUseRate:'',
                diskUseRate:'',
                diskClearUseRate:'',
                ikeSAMoreLicenseRate:'',
                ikeSAMoreLicenseClearRate:'',
                enbMoreLicenseRate:'',
                enbMoreLicenseClearRate:'',
            },
            alarmThresholdRules:{
                cpuUseRate:[{validator: validateAlarmValOrRegainVal,contrast:'cpuUseClearRate',type:'max'}],
                cpuUseClearRate:[{validator: validateAlarmValOrRegainVal,contrast:'cpuUseRate',type:'min'}],
                memoryUseRate:[{validator: validateAlarmValOrRegainVal,contrast:'memoryClearUseRate',type:'max'}],
                memoryClearUseRate:[{validator: validateAlarmValOrRegainVal,contrast:'memoryUseRate',type:'min'}],
                diskUseRate:[{validator: validateAlarmValOrRegainVal,contrast:'diskClearUseRate',type:'max'}],
                diskClearUseRate:[{validator: validateAlarmValOrRegainVal,contrast:'diskUseRate',type:'min'}],
                ikeSAMoreLicenseRate:[{validator: validateAlarmValOrRegainVal,contrast:'ikeSAMoreLicenseClearRate',type:'max'}],
                ikeSAMoreLicenseClearRate:[{validator: validateAlarmValOrRegainVal,contrast:'ikeSAMoreLicenseRate',type:'min'}],
                enbMoreLicenseRate:[{validator: validateAlarmValOrRegainVal,contrast:'enbMoreLicenseClearRate',type:'max'}],
                enbMoreLicenseClearRate:[{validator: validateAlarmValOrRegainVal,contrast:'enbMoreLicenseRate',type:'min'}],
            },
            loadingDialogShow:false,
            //告警详情
            ALARM_ID:'',
            ALARM_IDENTIFIER:'',
            ALARM_NAME:'',
            ADDITIONAL_INFORMATION:'',
            ADDITIONAL_TEXT:'',
            SPECIFIC_PROBLEM:'',
            ALARM_SERVERITY:'',
            EVENT_TYPE:'',
            NE_TYPE:'',
            EQUIP_INFO:'',
            DEAL_STATE:'',
            EVENT_TIME:'',
            UPD_TIME:'',
            DEAL_USER:'',
            DEAL_TIME:'',
            DEAL_MEMO:'',
            CLEAR_USER:'',
            ClEAR_TIME:'',
            SUGGESTION:'',
            confirmTipFlag: false,
            clearFlag: false,
		};
	},
	computed: {
		optBtnShow() {
			return writableMap['CODE_EGW'] == true;
		},
	},
	methods: {
		init(row,code,sn,status){
			var vm =this;
            vm.egwCode = code;
            vm.activeQueryParams.deviceCode = code;
            vm.historyQueryParams.deviceCode = code;
            vm.activeAlarmTableUrl = '${ctx}/fault/view/queryViewPageList.action';
            vm.historyAlarmTableUrl = '${ctx}/fault/view/queryViewPageList.action';
		},
		/**
        * 告警列表 点击更多操作出现菜单
        * @param row{object}   行数据
        * @param ev{object}   event数据
        */ 
        optActiveAlarmClick(row,ev){ // 操作项 
            var vm = this ,
                showClear = true,
                confirmDis = true;
            
            vm.rowDataAlarm = row;
            if(row.deal_state == '0' || row.deal_state == '2'){
                confirmDis = false;
            }
            vm.activeAlarmMenusData= [
                {label:'<%=rb.getString("XiangXi")%>',cls:"el-icon-operation-info el-icon",code:'detail'},
                {label:'<%=rb.getString("QueRenGaoJing")%>',cls:"el-icon-operation-confirm el-icon CODE_EGW hidden" ,code:'confirm'},
                {label:'<%=rb.getString("FanQueRenGaoJing")%>',cls:"el-icon-operation-unconfirm el-icon CODE_EGW hidden" ,code:'unConfirm',disable:!confirmDis},
                {label:'<%=rb.getString("QingChuGaoJing")%>',cls:"el-icon-operation-clear el-icon CODE_EGW hidden",code:'clear'},
            ];
            
            vm.$nextTick(function(){
                document.body.click();
                vm.$refs.activeAlarmMenus.show(ev);
            });
        },
        /**
        * 告警列表 点击更多操作出现菜单
        * @param row{object}   行数据
        * @param ev{object}   event数据
        */ 
        optHistoryAlarmClick(row,ev){ // 操作项 
            var vm = this ,
                showClear = true,
                confirmDis = true;
            
            vm.rowDataAlarm = row;
            if(row.deal_state == '0' || row.deal_state == '2'){
                confirmDis = false;
            }
            vm.historyAlarmMenusData= [
                {label:'<%=rb.getString("XiangXi")%>',cls:"el-icon-operation-info el-icon",code:'detail'},
                {label:'<%=rb.getString("QueRenGaoJing")%>',cls:"el-icon-operation-confirm el-icon CODE_EGW hidden" ,code:'confirm'},
                {label:'<%=rb.getString("FanQueRenGaoJing")%>',cls:"el-icon-operation-unconfirm el-icon CODE_EGW hidden" ,code:'unConfirm',disable:!confirmDis},
                {label:'<%=rb.getString("ShanChuGaoJing")%>',cls:"el-icon-operation-delete el-icon CODE_EGW hidden",code:'del'}
            ];
            
            vm.$nextTick(function(){
                document.body.click();
                vm.$refs.historyAlarmMenus.show(ev);
            });
        },
        /**
        * 告警菜单 点击事件
        * @param ev{object}   行数据
        */ 
        clickAlarmMenu(ev){ //单点击方法 --操作  
            var vm = this,
                codes = {
                    detail:vm.detailAlarmInfo,
                    confirm:vm.openConfirmDialog,
                    unConfirm:vm.unconfirmAlarm,
                    clear:vm.clearAlarm,
                    del:vm.deleteAlarm
                };
            if(codes[ev.code]){
                codes[ev.code](vm.rowDataAlarm)
            }
        },
        //点击页面其他地方菜单收起
        handerClose(){ 
            this.$refs.activeAlarmMenus.hide();
            this.$refs.historyAlarmMenus.hide();
        },
        // 告警详情
        detailAlarmInfo(row){
            var vm = this;
            axios.post('${ctx}/cell/fault/queryAlarmDetail.action',stringify({
                alarm_id: row.alarm_id,
                type: row.alarm_type,
                timeZone: timeZone
            })).then(function(response){
                var data = response.data;

                if(data){
                    vm.ALARM_ID = data.ALARM_ID;
                    vm.ALARM_IDENTIFIER = data.ALARM_IDENTIFIER;
                    vm.ALARM_NAME = data.ALARM_NAME;
                    vm.ADDITIONAL_INFORMATION = data.ADDITIONAL_INFORMATION;
                    vm.ADDITIONAL_TEXT = data.ADDITIONAL_TEXT;
                    vm.SPECIFIC_PROBLEM = data.SPECIFIC_PROBLEM;
                    vm.ALARM_SERVERITY = data.ALARM_SERVERITY;
                    vm.EVENT_TYPE = data.EVENT_TYPE;
                    vm.NE_TYPE = data.NE_TYPE;
                    vm.EQUIP_INFO = data.EQUIP_INFO;
                    vm.DEAL_STATE = row.deal_state;
                    vm.EVENT_TIME = data.EVENT_TIME;
                    vm.UPD_TIME = data.UPD_TIME;
                    vm.DEAL_USER = data.DEAL_USER;
                    vm.DEAL_TIME = data.DEAL_TIME;
                    vm.CLEAR_USER = data.CLEAR_USER;
                    vm.ClEAR_TIME = data.ClEAR_TIME;
                    vm.SUGGESTION = data.SUGGESTION;
                    vm.DEAL_MEMO = data.DEAL_MEMO;
                    // 1,3 已确认
                    if( data.DEAL_INT_STATE == '1' || data.DEAL_INT_STATE == '3'){
                        vm.confirmTipFlag = true
                    }
                    // 2,3 已清除
                    if( data.DEAL_INT_STATE == '3' || data.DEAL_INT_STATE == '2'){
                        vm.clearFlag = true
                    }else{
                        vm.clearFlag = false
                    }
                     vm.showAlarmInfo = true;
                }
            })
        },
        alarmInfoDialogClose(){
            var vm = this;

            vm.showAlarmInfo = false;
        },
        // 告警确认
        openConfirmDialog(row){
            var vm = this,
                status = row.deal_state;
            if(status == '1' || status == '3'){
                vm.alarmConfirmFlag = true;
            }else{
                vm.alarmConfirmFlag = false;
            }
            vm.showConfirmAlarmDialog = true;
            axios.post('${ctx}/cell/fault/queryAlarmDetail.action',stringify({
                alarm_id : vm.rowDataAlarm.alarm_id,
                type:vm.rowDataAlarm.alarm_type,
                timeZone:timeZone
            })).then(function(response){
                var data = response.data;
                vm.alarmConfirmForm.confirmUser = data.DEAL_USER;
                vm.alarmConfirmForm.confirmTime = data.DEAL_TIME;
                vm.alarmConfirmForm.description = data.DEAL_MEMO;
            }) 
        },
        // 告警确认提交
        confirmAlarmSubmit(){
            var vm = this,
                url = '${ctx}/cell/fault/confirmAlarm.action',   //确认告警 
                msg = '<%=rb.getString("QueRenGaoJingTiShi")%>';
            
            axios.post(url,stringify({
                alarm_id : vm.rowDataAlarm.alarm_id,
                small_cell_code: vm.rowDataAlarm.small_cell_code,
                type : vm.rowDataAlarm.alarm_type,
                text : vm.alarmConfirmForm.description
            })).then(function(response){
                var data = response.data;
                if(data["success"]){
                    vm.confirmAlarmDialogClose();
                    vm.$message.success( '<%=rb.getString("ChengGong")%>');
                    vm.$refs.activeAlarmTable.refresh();//表格刷新
                    vm.$refs.historyAlarmTable.refresh();//表格刷新
                }else{
                    vm.confirmAlarmDialogClose();
                    vm.$message.error(msg + '<%=rb.getString("TiShiShiBai")%>') //错误提示信息
                }
            }) 
        },
        // 告警反确认
        unconfirmAlarm(row){
            var vm = this;
            vm.$confirm('<%=rb.getString("QueDingQuXiaoGaoJingQueRen")%>','<%=rb.getString("QueRen")%>',{
                customClass:'warningConfirm',
                confirmButtonText:'<%=rb.getString("QueDing")%>',
                cancelButtonText:'<%=rb.getString("QuXiao")%>',
            }).then(function(){
                axios.post('${ctx}/cell/fault/cancelConfirmAlarm.action',stringify({
                    alarm_id : row.alarm_id,
                    type:row.alarm_type
                })).then(function(response){
                    var data = response.data;
                    if(data["success"]){
                        vm.$message.success('<%=rb.getString("ChengGong")%>');
                        vm.$refs.activeAlarmTable.refresh();//表格刷新
                        vm.$refs.historyAlarmTable.refresh();//表格刷新
                    }else{
                        vm.$message.error('<%=rb.getString("FanQueRenGaoJingTiShi")%><%=rb.getString("TiShiShiBai")%>') //错误提示信息
                    }
                })
            }).catch(function(){
                
            })
        },
        // 清除告警
        clearAlarm(){
            var vm = this;
            vm.showClearAlarmDialog = true;
        },
        // 清除告警提交事件
        clearAlarmSubmit(){
            var vm = this,
                url = '${ctx}/cell/fault/clearAlarm.action',   //确认告警 
                msg = '<%=rb.getString("QingChuGaoJingTiShi")%>';
            
            axios.post(url,stringify({
                alarm_id : vm.rowDataAlarm.alarm_id,
                small_cell_code: vm.rowDataAlarm.small_cell_code,
                type : vm.rowDataAlarm.alarm_type,
                text : vm.clearAlarmForm.description
            })).then(function(response){
                var data = response.data;
                if(data["success"]){
                    vm.clearAlarmDialogClose();
                    vm.$message.success( '<%=rb.getString("ChengGong")%>');
                    vm.$refs.activeAlarmTable.refresh();//表格刷新
                    vm.$refs.historyAlarmTable.refresh();//表格刷新
                }else{
                    vm.clearAlarmDialogClose();
                    vm.$message.error(msg + '<%=rb.getString("TiShiShiBai")%>') //错误提示信息
                }
            }) 
        },
        // 删除告警
        deleteAlarm(row){
            var vm = this;

            vm.$confirm('<%=rb.getString("QueRenShanChuGaoJing")%>','<%=rb.getString("QueRen")%>',{
                customClass:'warningConfirm',
                confirmButtonText:'<%=rb.getString("QueDing")%>',
                cancelButtonText:'<%=rb.getString("QuXiao")%>',
            }).then(function(){
                axios.post('${ctx}/cell/fault/clearHistoryAlarm.action',stringify({
                    alarm_id : row.alarm_id,
                    small_cell_code:row.small_cell_code,
                })).then(function(response){
                    var data = response.data;
                    if(data["success"]){
                        vm.$message.success('<%=rb.getString("ChengGong")%>')
                        vm.$refs.activeAlarmTable.refresh();//表格刷新
                        vm.$refs.historyAlarmTable.refresh();//表格刷新
                    }else{
                        vm.$message.error('<%=rb.getString("ShanChuGaoJingShiBai")%>') //错误提示信息 
                    }
                }) 
            }).catch(function(){
                
            })
        },
        // 告警详情弹窗打开成功事件
        openAlarmInfoDialogSuc(){
            var vm = this;
            eventBus.$emit('detail-info',vm.rowDataAlarm.alarm_type,vm.rowDataAlarm.alarm_id);
        },
        // 告警详情弹窗关闭
        closeAlarmInfoDialog(){
            var vm = this;
            vm.showAlarmInfo = false;
            vm.alarmInfoDialogUrl = '';
        },
        // 清除弹窗关闭事件
        clearAlarmDialogClose(){
            var vm = this;
            vm.showClearAlarmDialog = false;
            vm.clearAlarmForm.description = '';
        },
        // 确认弹窗关闭事件
        confirmAlarmDialogClose(){
            var vm = this,
                params = {
                    confirmUser:'',
                    confirmTime:'',
                    description:''
                };
            vm.showConfirmAlarmDialog = false;
            Object.assign(vm.alarmConfirmForm,params);
        },
        // 告警设置 打开告警设置弹窗事件
		egwAlarmSetting(){
            var vm = this;
            vm.getCommonConfigInfo();
            vm.loadingDialogShow = true;
            vm.showAlarmThresholdDialog = true;
        },
        getCommonConfigInfo(){
            var vm = this,
                code = vm.egwCode,
                params={
                    egwCode:code
                };
            axios.post('${ctx}/egw/config/getAlarmThreshold.action',stringify(params)).then(function(response){
                var data = response.data;
                if(data){
                    Object.keys(vm.alarmThresholdForm).forEach(function(key){
                        if(data[key]){
                            vm.alarmThresholdForm[key] = data[key];
                        }
                    });
                    initForm(vm.$refs.alarmThresholdForm);
                    vm.loadingDialogShow = false;
                }else{
                    vm.loadingDialogShow = false;
                }
            }).catch(function(error){});

        },
        // 告警值与恢复值 相互校验
        commonConfigValid(val){
            var vm = this;
            vm.$refs.alarmThresholdForm.validateField(val);
        },
        // 验证输入的是否是数字
        isNumeric(str) {
            if(str.length==0){
                return false;
            }
            for(var i=0;i<str.length;i++){
                if(str.charAt(i)<"0" || str.charAt(i)>"9"){
                    return false;
                }
            }
            return true;  
        },
         // 告警设置 提交
        alarmThresholdSubmit(){
            var vm = this,
                urls = '${ctx}/egw/config/setAlarmThreshold.action',
                params = {
                    egwCode:vm.egwCode
                },
                isChanged = isFormChanged(vm.$refs.alarmThresholdForm);
            if(!isChanged){
                showMsg('prompt_msg','<%=rb.getString("CanShuZhiMeiYouBianHua")%>');
                return;
            }
            vm.$refs.alarmThresholdForm.fields.map(function(field){
                if(Array.isArray(field.fieldValue)){
                    
                    var vList = field.fieldValue.map(function(item){return item}),
                        oList = (field.reinitialValue||[]).map(function(item){return item}),
                        val = JSON.stringify(vList.sort()),
                        orVal = JSON.stringify(oList.sort());

                    if(val != orVal) {
                        var editList=[],subList=[];
                        if(val != orVal) {
                            editList.push(field.prop);
                        }
                    };
                }else{
                    if(vm.isNull(field.fieldValue) && vm.isNull(field.reinitialValue)){
                            
                    }else if(field.fieldValue != field.reinitialValue) {
                        params[field.prop] = field.fieldValue;
                    };
                }
            });
            vm.$refs["alarmThresholdForm"].validate( valid => {
                if(valid){
                    axios.post(urls,stringify(params)).then(res=>{
                        var data = res.data;
                        if(data["success"]){
                            vm.$message({
                                message:'<%=rb.getString("ChengGong")%>',
                                type:'success',
                            })
                            vm.showAlarmThresholdDialog = false;
                        }else{
                            vm.$message.error(data["message"])
                        }
                    })
                }else{
                    return false
                }
            }) 
        },
        // 告警设置弹窗 关闭事件
        alarmThresholdDialogClose(){
            var vm = this,
                params = {
                    cpuUseRate:'',
                    cpuUseClearRate:'',
                    memoryUseRate:'',
                    memoryClearUseRate:'',
                    diskUseRate:'',
                    diskClearUseRate:'',
                    ikeSAMoreLicenseRate:'',
                    ikeSAMoreLicenseClearRate:'',
                    enbMoreLicenseRate:'',
                    enbMoreLicenseClearRate:'',
                };
            Object.assign(vm.alarmThresholdForm,params);
            vm.$refs["alarmThresholdForm"].clearValidate();
        },
        // 判断是否为空
        isNull(val){
            if(val==undefined || val == null || val =="") return true;
            else return false;
        },
	},
	mounted() {
		eventBus.$off("egw-data").$on("egw-data",this.init)
	}
});

</script>
