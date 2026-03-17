<%@ page import="java.util.Locale"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<style>
#gnbDetailPages .alarmMinorIcon::before,
#gnbAlarmPage .alarmMinorIcon::before{
	color: #FFDA41;
	font-size: 18px;
}
#gnbDetailPages .alarmMajorIcon::before,
#gnbAlarmPage .alarmMajorIcon::before{
	color: #FF973E;
	font-size: 18px;
}
#gnbDetailPages .alarmCriticalIcon::before,
#gnbAlarmPage .alarmCriticalIcon::before{
	color: #FC5959;
	font-size: 18px;
}
#gnbDetailPages .alarmWarningIcon::before,
#gnbAlarmPage .alarmWarningIcon::before{
	color: #60BEFC;
	font-size: 18px;
}
#gnbAlarmPage .disabledClass {
	cursor: not-allowed !important; 
	opacity: 0.4;
	font-size: 16px; 
}
#gnbAlarmPage .defaultClass {
	cursor: pointer; 
	font-size: 16px; 
	color: rgba(0, 0, 0, 0.8);	
}
#gnbAlarmPage .el-icon-operation-more-circle:hover, .el-icon-operation-more-circle:active {
	color: var(--main-color) !important; 
}
#gnbAlarmPage .alarmItemTableBoxCls {
    height: 50%;
}
#gnbDetailPages label{
	color: #7A7992;
	margin-right:5px;
	text-align:left;
	display:inline-block;
	width:200px;
}
#gnbDetailPages span{
	color: rgba(0, 0, 0, 0.8);
	width: 730px;
	word-wrap: break-word;
}
#gnbDetailPages div{
	margin-bottom:20px;
	font-size:12px;
	display: flex;
	width: 940px;
}
</style>

<div id="gnbAlarmPage" class='commonWarp' style='background: #FFFFFF;width: 100%; height: 100%;'>
    <div class="alarmItemTableBoxCls">
        <el-ctable ref="gnbActiveAlarmTable" id="gnbActiveAlarmTable" :url="gnbAlarmTableUrl" :query-params="queryParams" pagination="true" :time="6" :rownumber=true :row-key="'serial_number'">
            <template slot="toolbar">
                <div class='toolbarHeadBtnBoxCls'>
                    <div class='commonText14' style='margin-left: 20px;'><%=rb.getString("HuoDongGaoJingLieBiao")%></div>
                 </div>
            </template>
            <el-table-column label='' width="40">
                <template slot-scope="scope">
                     <div class="el-icon el-icon-operation-more-circle" @click="optActiveAlarmClick(scope.row,event)" v-clickoutside="handerClose"></div>
                 </template>
            </el-table-column>
            <el-table-column label='<%=rb.getString("XuHao")%>' prop="alarm_id" sortable width="120"></el-table-column>
            <el-table-column label='<%=rb.getString("GaoJingJiBie")%>' prop="alarm_serverity_value" sortable>
                <template slot-scope="scope">
                    <div v-if="scope.row.alarm_serverity_value == 'Minor'">
                        <span class="el-icon el-icon-status-alarm alarmMinorIcon" style="margin-right:5px;"></span><%=rb.getString("CiYaoGaoJing")%>
                    </div>
                    <div v-else-if="scope.row.alarm_serverity_value == 'Major'">
                        <span class="el-icon el-icon-status-alarm alarmMajorIcon" style="margin-right:5px;"></span><%=rb.getString("ZhuYaoGaoJing")%>
                    </div>
                    <div v-else-if="scope.row.alarm_serverity_value == 'Critical'">
                        <span class="el-icon el-icon-status-alarm alarmCriticalIcon" style="margin-right:5px;"></span><%=rb.getString("JinJiGaoJing")%>
                    </div>
                    <div v-else-if="scope.row.alarm_serverity_value == 'Warning'">
                        <span class="el-icon el-icon-status-alarm alarmWarningIcon" style="margin-right:5px;"></span><%=rb.getString("JingGaoGaoJing")%>
                    </div>
                </template>
            </el-table-column>
            <el-table-column label='<%=rb.getString("GaoJingWeiYiBiaoZhi")%>' prop="alarm_identifier"></el-table-column>
            <el-table-column label='<%=rb.getString("KeNengYuanYin")%>' prop="alarm_name" show-overflow-tooltip></el-table-column>
            <el-table-column label='<%=rb.getString("GaoJingZhuangTai")%>' prop="deal_state" show-overflow-tooltip sortable width="220">
                <template slot-scope="scope">
                    <div v-if="scope.row.deal_state == '0'" style="display:flex;align-items: center;">
                        <span class="el-icon el-icon-status-unconfirmActive redIcon" style="margin-right:5px; font-size: 22px;"></span><%=rb.getString("WeiQueRenWeiQingChu")%>
                    </div>
                    <div v-else-if="scope.row.deal_state == '1'" style="display:flex;align-items: center;">
                        <span class="el-icon el-icon-status-confirmActive redIcon" style="margin-right:5px;font-size: 22px;"></span><%=rb.getString("YiQueRenWeiQingChu")%>
                    </div>
                    <div v-if="scope.row.deal_state == '2'" style="display:flex;align-items: center;">
                        <span class="el-icon el-icon-status-unconfirmActive greenIcon" style="margin-right:5px;font-size: 22px;"></span><%=rb.getString("WeiQueRenYiQingChu")%>
                    </div>
                    <div v-else-if="scope.row.deal_state == '3'" style="display:flex;align-items: center;">
                        <span class="el-icon el-icon-status-confirmActive greenIcon" style="margin-right:5px;font-size: 22px;"></span><%=rb.getString("YiQueRenYiQingChu")%>
                    </div>
                </template>
            </el-table-column>
            <el-table-column label='<%=rb.getString("GuZhangShiJian")%>' prop="event_time" sortable></el-table-column>
        </el-ctable>
        <el-cmenu ref="activeAlarmMenus" :data="activeAlarmMenusData" @click="clickAlarmMenu"></el-cmenu>
    </div>
    <!--历史告警列表 -->
    <div class="alarmItemTableBoxCls" style='border-top: 1px solid #D5DCEC;'>
        <el-ctable ref="gnbHistoryAlarmTable" id="gnbHistoryAlarmTable" :url="gnbHistoryAlarmTableUrl" :query-params="historyQueryParams" pagination="true" :time="6" :rownumber=true :row-key="'serial_number'">
            <template slot="toolbar">
                <div class='toolbarHeadBtnBoxCls'>
                    <div class='commonText14' style='margin-left: 20px;'><%=rb.getString("LiShiGaoJingLieBiao")%></div>
                 </div>
            </template>
            <el-table-column label='' width="40">
                <template slot-scope="scope">
                     <div class="el-icon el-icon-operation-more-circle" @click="optHistoryAlarmClick(scope.row,event)" v-clickoutside="handerClose"></div>
                 </template>
            </el-table-column>
            <el-table-column label='<%=rb.getString("XuHao")%>' prop="alarm_id" sortable width="120"></el-table-column>
            <el-table-column label='<%=rb.getString("GaoJingJiBie")%>' prop="alarm_serverity_value" sortable>
                <template slot-scope="scope">
                    <div v-if="scope.row.alarm_serverity_value == 'Minor'">
                        <span class="el-icon el-icon-status-alarm alarmMinorIcon" style="margin-right:5px;"></span><%=rb.getString("CiYaoGaoJing")%>
                    </div>
                    <div v-else-if="scope.row.alarm_serverity_value == 'Major'">
                        <span class="el-icon el-icon-status-alarm alarmMajorIcon" style="margin-right:5px;"></span><%=rb.getString("ZhuYaoGaoJing")%>
                    </div>
                    <div v-else-if="scope.row.alarm_serverity_value == 'Critical'">
                        <span class="el-icon el-icon-status-alarm alarmCriticalIcon" style="margin-right:5px;"></span><%=rb.getString("JinJiGaoJing")%>
                    </div>
                    <div v-else-if="scope.row.alarm_serverity_value == 'Warning'">
                        <span class="el-icon el-icon-status-alarm alarmWarningIcon" style="margin-right:5px;"></span><%=rb.getString("JingGaoGaoJing")%>
                    </div>
                </template>
            </el-table-column>
            <el-table-column label='<%=rb.getString("GaoJingWeiYiBiaoZhi")%>' prop="alarm_identifier"></el-table-column>
            <el-table-column label='<%=rb.getString("KeNengYuanYin")%>' prop="alarm_name" show-overflow-tooltip></el-table-column>
            <el-table-column label='<%=rb.getString("GaoJingZhuangTai")%>' prop="deal_state" show-overflow-tooltip sortable width="220">
                <template slot-scope="scope">
                    <div v-if="scope.row.deal_state == '0'" style="display:flex;align-items: center;">
                        <span class="el-icon el-icon-status-unconfirmActive redIcon" style="margin-right:5px;font-size: 22px;"></span><%=rb.getString("WeiQueRenWeiQingChu")%>
                    </div>
                    <div v-else-if="scope.row.deal_state == '1'" style="display:flex;align-items: center;">
                        <span class="el-icon el-icon-status-confirmActive redIcon" style="margin-right:5px;font-size: 22px;"></span><%=rb.getString("YiQueRenWeiQingChu")%>
                    </div>
                    <div v-if="scope.row.deal_state == '2'" style="display:flex;align-items: center;">
                        <span class="el-icon el-icon-status-unconfirmActive greenIcon" style="margin-right:5px;font-size: 22px;"></span><%=rb.getString("WeiQueRenYiQingChu")%>
                    </div>
                    <div v-else-if="scope.row.deal_state == '3'" style="display:flex;align-items: center;">
                        <span class="el-icon el-icon-status-confirmActive greenIcon" style="margin-right:5px;font-size: 22px;"></span><%=rb.getString("YiQueRenYiQingChu")%>
                    </div>
                </template>
            </el-table-column>
            <el-table-column label='<%=rb.getString("GuZhangShiJian")%>' prop="event_time" sortable></el-table-column>
        </el-ctable>
        <el-cmenu ref="historyAlarmMenus" :data="historyAlarmMenusData" @click="clickAlarmMenu"></el-cmenu>
    </div>
         
	<!--告警确认弹窗-->
	<el-dialog :title="'<%=rb.getString("QueRenGaoJing")%>'" :visible.sync="showConfirmAlarmDialog" :close-on-click-modal="false" @close="confirmAlarmDialogClose" :append-to-body="true" top="15vh"  width="500">
		<el-form :model="confirmForm" ref="confirmForm" label-position="top">
			<el-form-item v-if="confirmFlag" label="<%=rb.getString("QueRenRen")%>" prop="confirmUser">
				<el-input :disabled="true" v-model="confirmForm.confirmUser"></el-input>
			</el-form-item>
			<el-form-item v-if="confirmFlag" label="<%=rb.getString("QueRenShiJian")%>" prop="confirmTime">
				<el-input :disabled="true" v-model="confirmForm.confirmTime">
			</el-form-item>
			<el-form-item label="<%=rb.getString("MiaoShu")%>" prop="description">
				<el-input type="textarea" :rows="3" maxlength="500" v-model="confirmForm.description">
			</el-form-item>
		</el-form>
		<span slot="footer" class="dialog-footer">
			<div class="buttonGroup">
				<el-button type="primary" @click="confirmAlarmSubmit"><%=rb.getString("QueDing")%></el-button>
				<el-button @click="confirmAlarmDialogClose"><%=rb.getString("QuXiao")%></el-button>
			</div>	
		</span>
	</el-dialog>
	
	<!--告警清除弹窗-->
	<el-dialog :title="'<%=rb.getString("QueRen")%>'" :visible.sync="showClearAlarmDialog" :close-on-click-modal="false" @close="clearAlarmDialogClose" :append-to-body="true" top="15vh" width="500" >
		<el-form  :model="clearForm" ref="clearForm" label-position="top">
			<el-form-item>
				<%=rb.getString("QueRenQingChuGaoJing")%>
			</el-form-item>
			<el-form-item label="<%=rb.getString("MiaoShu")%>" prop="description">
				<el-input type="textarea" :rows="3" v-model="clearForm.description">
			</el-form-item>
		</el-form>
		<span slot="footer" class="dialog-footer">
			<div class="buttonGroup">
				<el-button type="primary" @click="clearAlarmSubmit"><%=rb.getString("QueDing")%></el-button>
				<el-button @click="clearAlarmDialogClose"><%=rb.getString("QuXiao")%></el-button>
			</div>	
		</span>
	</el-dialog>
	<!--告警详情弹窗-->
	<el-dialog :title="'<%=rb.getString("XiangXiXinXi")%>'" :visible.sync="showAlarmInfo" :close-on-click-modal="false" @close="alarmInfoDialogClose" :append-to-body="true" top="15vh" width="1000" >
        <div id='gnbDetailPages'>
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
</div>

<script type="text/javascript">
	var gnbAlarmPageVue = new Vue({
	    el: '#gnbAlarmPage',
	    data() {
	    	return {
	    		activeAlarmMenusData: [],
	    		queryParams: {
	    			timeZone: timeZone,
	    			alarmType: 'ACTIVE',
					queryType: 'View',
					alarmServerity: '31001,31002,31003,31004',
					deviceCode: '',
					neType: 'GNB'
	            }, 
				gnbAlarmTableUrl: '',

				rowData: [],
				showConfirmAlarmDialog: false,
				confirmFlag: false,
				confirmForm:{
					confirmUser: '',
					confirmTime: '',
					description: '',
				},
				showClearAlarmDialog: false,
				clearForm: {
					description: '',
				},
				rowDataAlarm: [],  // 右侧告警列表点击数据
				rowDataClearAlarm: [],
				//历史告警列表
				activeHistoryFlag: false,
				gnbHistoryAlarmTableUrl: '',
				historyQueryParams:{
                    timeZone:timeZone,
                    alarmType:'HISTORY',
                    queryType:'View',
                    alarmServerity:'31001,31002,31003,31004',
                    deviceCode:'',
                    neType:'GNB'
                },
                historyAlarmMenusData: [],
                showAlarmInfo: false,
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
	    	}
	    },
	    methods: {
			init(row){
				var vm = this;
				vm.queryParams.deviceCode = row.small_cell_code;
				vm.gnbAlarmTableUrl = '${ctx}/fault/view/queryViewPageList.action';
				vm.historyQueryParams.deviceCode = row.small_cell_code;
				vm.gnbHistoryAlarmTableUrl = '${ctx}/fault/view/queryViewPageList.action';
			},
			/**
			* 告警列表 点击更多操作出现菜单
			* @param row{object}   行数据
			* @param ev{object}   event数据
			*/ 
			optActiveAlarmClick(row,ev){ // 操作项
				var vm = this,
					showClear= true,
					confirmDis = true;
				
				vm.rowDataAlarm = row;
				vm.activeHistoryFlag = true;
				if(row.deal_state == '0' || row.deal_state == '2'){
					confirmDis = false;
				}
				vm.activeAlarmMenusData= [
				    {label:'<%=rb.getString("XinXi")%>',cls:"el-icon-operation-info el-icon",code:'detail'},
					{label:'<%=rb.getString("QueRenGaoJing")%>',cls:"el-icon-operation-confirm el-icon CODE_ALARM_VIEW hidden" ,code:'confirm'},
					{label:'<%=rb.getString("FanQueRenGaoJing")%>',cls:"el-icon-operation-unconfirm el-icon CODE_ALARM_VIEW hidden" ,code:'unConfirm',disable:!confirmDis},
					{label:'<%=rb.getString("QingChuGaoJing")%>',cls:"el-icon-operation-clear el-icon CODE_ALARM_VIEW hidden",code:'clear'},
				];
				
				vm.$nextTick(function(){
					document.body.click();
					vm.$refs.activeAlarmMenus.show(ev);
				});
			},
			//--------------------------------------历史告警
            optHistoryAlarmClick(row,ev){ // 操作项
                var vm = this,
                    showClear = true,
                    confirmDis = true;

                vm.rowDataAlarm = row;
                vm.activeHistoryFlag = false;
                if(row.deal_state == '0' || row.deal_state == '2'){
                     confirmDis = false;
                }
                vm.historyAlarmMenusData= [
                    {label:'<%=rb.getString("XinXi")%>',cls:"el-icon-operation-info el-icon",code:'detail'},
                    {label:'<%=rb.getString("QueRenGaoJing")%>',cls:"el-icon-operation-confirm el-icon CODE_ALARM_VIEW hidden" ,code:'confirm'},
                    {label:'<%=rb.getString("FanQueRenGaoJing")%>',cls:"el-icon-operation-unconfirm el-icon CODE_ALARM_VIEW hidden" ,code:'unConfirm',disable:!confirmDis},
                    {label:'<%=rb.getString("ShanChuGaoJing")%>',cls:"el-icon-operation-delete el-icon CODE_ALARM_VIEW hidden",code:'del'}
                ];

                vm.$nextTick(function(){
                    document.body.click();
                    vm.$refs.historyAlarmMenus.show(ev);
                });
              },
			//点击页面其他地方菜单收起
			handerClose(){ 
				this.$refs.activeAlarmMenus.hide();
				this.$refs.historyAlarmMenus.hide();
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
				var vm = this, status = row.deal_state;
				
				if(status == '1' || status == '3'){
					vm.confirmFlag = true;
				}else{
					vm.confirmFlag = false;
				}
				vm.rowDataAlarm = row;
				vm.showConfirmAlarmDialog = true;
				
				axios.post('${ctx}/cell/fault/queryAlarmDetail.action',stringify({
					alarm_id: row.alarm_id,
					type: row.alarm_type,
					timeZone: timeZone
				})).then(function(response){
					var data = response.data;
					
					if(data){
						vm.confirmForm.confirmUser = data.DEAL_USER;
						vm.confirmForm.confirmTime = data.DEAL_TIME;
						vm.confirmForm.description = data.DEAL_MEMO;
					}
				}) 
			},
			// 告警确认提交
			confirmAlarmSubmit(){
				var vm = this,
					url = '${ctx}/cell/fault/confirmAlarm.action',
					msg = '<%=rb.getString("QueRenGaoJingTiShi")%>';
				
				axios.post(url,stringify({
					alarm_id: vm.rowDataAlarm.alarm_id,
					small_cell_code: vm.rowDataAlarm.small_cell_code,
					type: vm.rowDataAlarm.alarm_type,
					text: vm.confirmForm.description
				})).then(function(response){
					var data = response.data;
					
					if(data["success"]){
						vm.confirmAlarmDialogClose();
						vm.$message.success('<%=rb.getString("ChengGong")%>');
						if(vm.activeHistoryFlag  == true){
						    vm.$refs.gnbActiveAlarmTable.refresh();
						}else{
						    vm.$refs.gnbHistoryAlarmTable.refresh();
						}
					}else{
						vm.confirmAlarmDialogClose();
						vm.$message.error(msg + '<%=rb.getString("TiShiShiBai")%>');
					}
				})
				
			},
			// 告警反确认
			unconfirmAlarm(row){
				var vm = this;
				vm.$confirm('<%=rb.getString("QueDingQuXiaoGaoJingQueRen")%>','<%=rb.getString("QueRen")%>',{
					customClass: 'warningConfirm',
					confirmButtonText: '<%=rb.getString("QueDing")%>',
					cancelButtonText: '<%=rb.getString("QuXiao")%>',
				}).then(function(){
					axios.post('${ctx}/cell/fault/cancelConfirmAlarm.action',stringify({
						alarm_id: row.alarm_id,
						type: row.alarm_type
					})).then(function(response){
						var data = response.data;
						
						if(data["success"]){
							vm.$message.success('<%=rb.getString("ChengGong")%>');
							if(vm.activeHistoryFlag  == true){
                                vm.$refs.gnbActiveAlarmTable.refresh();
                            }else{
                                vm.$refs.gnbHistoryAlarmTable.refresh();
                            }
						}else{
							vm.$message.error('<%=rb.getString("FanQueRenGaoJingTiShi")%><%=rb.getString("TiShiShiBai")%>');
						}
					})
				}).catch(function(){
					
				})
			},
			// 确认弹窗关闭事件
			confirmAlarmDialogClose(){
				var vm = this,
					params = {
						confirmUser: '',
						confirmTime: '',
						description: ''
					};
				vm.showConfirmAlarmDialog = false;
				Object.assign(vm.confirmForm,params);
			},
			// 清除告警
			clearAlarm(row){
				var vm = this;
				
				vm.rowDataClearAlarm = row;
				vm.showClearAlarmDialog = true;
			},
			// 清除告警提交事件
			clearAlarmSubmit(){
				var vm = this,
					url = '${ctx}/cell/fault/clearAlarm.action',
					msg = '<%=rb.getString("QingChuGaoJingTiShi")%>';
				
				axios.post(url,stringify({
					alarm_id: vm.rowDataClearAlarm.alarm_id,
					small_cell_code: vm.rowDataClearAlarm.small_cell_code,
					type: vm.rowDataClearAlarm.alarm_type,
					text: vm.clearForm.description
				})).then(function(response){
					var data = response.data;
					
					if(data["success"]){
						vm.clearAlarmDialogClose();
						vm.$message.success('<%=rb.getString("ChengGong")%>');
						vm.$refs.gnbActiveAlarmTable.refresh();
					}else{
						vm.clearAlarmDialogClose();
						vm.$message.error(msg + '<%=rb.getString("TiShiShiBai")%>');
					}
				}) 
			},
			// 清除弹窗关闭事件
			clearAlarmDialogClose(){
				var vm = this;
				
				vm.showClearAlarmDialog = false;
				vm.clearForm.description = '';
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
                            vm.$refs.gnbHistoryAlarmTable.refresh();
                        }else{
                            vm.$message.error('<%=rb.getString("ShanChuGaoJingShiBai")%>')
                        }
                    })
                }).catch(function(){

                })
            }
	    },
	    mounted(){
	    	eventBus.$off("gnb-data").$on("gnb-data",this.init)
	    }
	});
</script> 	
