<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	#distributeDetail{
		width:100%;
		height:100%;
		display:flex;
		flex-direction:column;
		flex:1 1 auto
	}
	#distributeDetail .el-icon-status-alarm:before{
		color:#F2B354;
	}
	#distributeDetail .el-icon-status-upgrading:before{
		color:#4D84FF;
	}
	#distributeDetail .el-icon-status-upgrade-success:before{
		color:#67D972;
	}
	#distributeDetail .el-icon-status-enable:before{
		color:#67D972;
	}
	#distributeDetail .el-icon-status-disable:before{
		color:#CFCFCF;
	}
	#distributeDetail .el-icon-status-conn-on:before{
		color:#67D972;
	}
	.boardCardOptCls{
		display: flex;
		align-items: center;
		justify-content: space-between;
	}
	#distributeDetail div{
		box-sizing: border-box;
	}
	#distributeDetail .el-collapse-item__header{
		border-bottom:1px solid #fff;
	}
	#distributeDetail .el-collapse-item__arrow{
		position:absolute;
		left:20px;
		top:0px;
	}
	#distributeDetail .el-collapse-item{
		position:relative;
		background:#FFF;
	}
	#distributeDetail .el-collapse-item__arrow.is-active{
		transform:rotate(0deg);
	}
	#distributeDetail .el-collapse{
		border-top:1px solid #fff;
		border-bottom:1px solid #fff;
		height: calc(100% - 30px);
	}
	#distributeDetail .el-collapse-item__wrap{
		border-bottom:1px solid #fff;
		padding-left:unset;
	}
	#distributeDetail .el-collapse-item__header{
		max-width:300px;
	}
	#distributeDetail .rightContentCls{
		margin-left: 65px;
		height: 180px;
	}
	#distributeDetail .noCardTableContentCls{
		margin-left: 65px;
		height: 280px;
	}
	#editBoardCardDialog .el-radio{
		width: 60px;
	}
	.statusDiv {
		display:inline-block;
		margin:0 20px;
	}
	.borderPage {
		background:#FFF;
	}
</style>
<div id="distributeDetail" class="borderPage" >
	<div class="el-card__body" style='height:calc(100% - 50px);overflow:auto;padding:0;display:flex;flex-direction:column'>
		<el-collapse v-model="stationCollapse" style="padding:0px 0px 0px 20px;">
			<el-collapse-item name="BC" v-if="boardCardShow">
				<template slot='title'>
					<p style="display:inline-block;margin-left:40px;">
						<span class="title-icon" style="vertical-align:sub"></span>
						<span style="font-size:14px;font-weight:bold"><%=rb.getString("BanKa") %></span>
					</p>
				</template>
				<div :class="boardCardShow == true ? 'rightContentCls' : 'noCardTableContentCls' ">
					<!--:url="boardCardUrl" :data="boardCardData" -->
					<div style='height:100%;margin:10px 0px;'>
						<el-ctable id="boardCardTable" ref="boardCardTable" :url="boardCardUrl"  :query-params="queryParams"
							:height="height" :time="6" :pagination="false" rownumber="true" style="margin-right:100px;border:1px solid #E9E9E9;">
							<el-table-column v-if="optBtnShow" label="" width="70" prop="">
								<template slot-scope="scope">
									<div v-if="scope.row.cardStatus == '1'" class="boardCardOptCls">
										<span class="el-icon el-icon-operation-edit" @click="editBoardCard(scope.row)"></span>
										<span class="el-icon el-icon-operation-reboot" @click="rebootBoardCard(scope.row)"></span>
									</div>
									<div v-if="scope.row.cardStatus == '0'" class="boardCardOptCls">
										<span class="el-icon el-icon-operation-edit disabled"></span>
										<span class="el-icon el-icon-operation-reboot disabled"></span>
									</div>
								</template>
							</el-table-column>
							<el-table-column label="<%=rb.getString("BanKaID") %>" prop="cardId"></el-table-column>
							<el-table-column label="<%=rb.getString("BanKaSN") %>" prop="cardSn"></el-table-column>
							<el-table-column label="<%=rb.getString("BanKaZhuangTai") %>" prop="cardStatus">
								<template slot-scope="scope">
									<div v-if="scope.row.cardStatus == '1'">
										<span class='el-icon el-icon-status-conn-on' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("ZaiWei") %></span>
									</div>
									<div v-if="scope.row.cardStatus == '0'">
										<span class='el-icon el-icon-status-conn-off' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("BuZaiWei") %></span>
									</div>
								</template>
							</el-table-column>
							<el-table-column label="<%=rb.getString("XiaoQuGeShu") %>" prop="cellNum"></el-table-column>
							<el-table-column label="<%=rb.getString("JiZhanZhiShi") %>" prop="duplexMode"></el-table-column>
						</el-ctable>
					</div>
				</div>
			</el-collapse-item>
			
			<el-collapse-item name="Statistic">
				<template slot='title'>
					<p style="display:inline-block;margin-left:40px;">
						<span class="title-icon" style="vertical-align:sub"></span>
						<span style="font-size:14px;font-weight:bold">Status</span>
					</p>
				</template>
				<div :class="boardCardShow == true ? 'rightContentCls' : 'noCardTableContentCls' ">
					<div style='height:100%;margin:10px 0px;'>
						<el-ctable id="statusTable" ref="statusTable" :data="statisticData" :query-params="queryParamsEUAndRU"
							:height="height" :pagination="false" rownumber="true" style="margin-right:100px;border:1px solid #E9E9E9;">
							<template slot="toolbar">
								<div style='display: flex;'>
									<div class="statusDiv">EU <%=rb.getString("ZhengChang") %> :
										<span> {{statisticCount.euOnlineCount}} </span>  / <span> {{statisticCount.euTotalCount}} </span>
									</div>
									<div class="statusDiv">RU <%=rb.getString("ZhengChang") %> :
										<span> {{statisticCount.ruOnlineCount}} </span>  / <span> {{statisticCount.ruTotalCount}} </span>
									</div>
									<el-query type="normal" @query="euAndRuQuery" placeholder='<%=rb.getString("EUJiZhanBianMa")%>/<%=rb.getString("RUJiZhanBianMa")%>'></el-query>
								</div>
							</template>
							
							<el-table-column label="<%=rb.getString("EUJiZhanBianMa") %>" prop="euSn"></el-table-column>
							<el-table-column label="EU <%=rb.getString("ZhuangTai") %>" prop="euStatus">
								<template slot-scope="scope">
									<div v-if="scope.row.euStatus == '1'">
										<span class='el-icon el-icon-status-conn-on' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("ZhengChang") %></span>
									</div>
									<div v-if="scope.row.euStatus == '2'">
										<span class='el-icon el-icon-status-conn-off' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("LiXian") %></span>
									</div>
									<div v-if="scope.row.euStatus == '3'">
										<span class='el-icon el-icon-status-alarm' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("GaoJingGuanLi") %></span>
									</div>
									<div v-if="scope.row.euStatus == '4'">
										<span class='el-icon el-icon-status-upgrading' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("ShengJiZhong") %></span>
									</div>
									<div v-if="scope.row.euStatus == '5'">
										<span class='el-icon el-icon-status-upgrade-success' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("ShengJi") %><%=rb.getString("ChengGong") %></span>
									</div>
									<!-- 只针对去重数据中 将eu 状态值置空-->
									<div v-if="scope.row.euStatus == '6'"> </div>
									<div v-if="scope.row.euStatus == null || scope.row.euStatus == ''">
										--
									</div>
								</template>
							</el-table-column>
							<el-table-column label="<%=rb.getString("RUJiZhanBianMa") %>" prop="ruSn">
								<template slot-scope="scope">
									<div v-if="scope.row.ruSn">
										{{scope.row.ruSn}}
									</div>
									<div v-else>
										--
									</div>
								</template>
							</el-table-column>
							<el-table-column label="RU <%=rb.getString("ZhuangTai") %>" prop="ruStatus">
								<template slot-scope="scope">
									<div v-if="scope.row.ruStatus == '1'">
										<span class='el-icon el-icon-status-conn-on' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("ZhengChang") %></span>
									</div>
									<div v-if="scope.row.ruStatus == '2'">
										<span class='el-icon el-icon-status-conn-off' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("LiXian") %></span>
									</div>
									<div v-if="scope.row.ruStatus == '3'">
										<span class='el-icon el-icon-status-alarm' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("GaoJingGuanLi") %></span>
									</div>
									<div v-if="scope.row.ruStatus == '4'">
										<span class='el-icon el-icon-status-upgrading' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("ShengJiZhong") %></span>
									</div>
									<div v-if="scope.row.ruStatus == '5'">
										<span class='el-icon el-icon-status-upgrade-success' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("ShengJi") %><%=rb.getString("ChengGong") %></span>
									</div>
									<div v-if="scope.row.ruStatus == null || scope.row.ruStatus == ''">
										--
									</div>
								</template>
							</el-table-column>
						</el-ctable>
					</div>
				</div>
			</el-collapse-item>
			
			
			<el-collapse-item name="EU">
				<template slot='title'>
					<p style="display:inline-block;margin-left:40px;">
						<span class="title-icon" style="vertical-align:sub"></span>
						<span style="font-size:14px;font-weight:bold">EU <%=rb.getString("SheBeiLieBiao") %></span>
					</p>
				</template>
				<div :class="boardCardShow == true ? 'rightContentCls' : 'noCardTableContentCls' ">
					<div style='height:100%;margin:10px 0px;'>
						<el-ctable id="euTable" ref="euTable" :url="euUrl"  :query-params="queryParams"
							:height="height" pagination="true" rownumber="true" style="margin-right:100px;border:1px solid #E9E9E9;">
							<el-table-column v-if="optBtnShow" label="" width="30" prop="" class-name="no-text-tips">
								<template slot-scope="scope">
									<div class="el-icon el-icon-operation-more" @click="optClick(scope.row,event)" v-clickoutside="handerClose" style="cursor: pointer;"></div>
								</template>
							</el-table-column>
							<el-table-column label="<%=rb.getString("LuYouSuoYin") %>" prop="route_index" sortable="true"></el-table-column>
							<el-table-column label="<%=rb.getString("BanKaID") %>" prop="card_id" v-if="boardCardShow"></el-table-column>
							<el-table-column label="<%=rb.getString("ZhuangTai") %>" prop="status">
								<template slot-scope="scope">
									<div v-if="scope.row.status == '1'">
										<span class='el-icon el-icon-status-conn-on' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("ZhengChang") %></span>
									</div>
									<div v-if="scope.row.status == '2'">
										<span class='el-icon el-icon-status-conn-off' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("LiXian") %></span>
									</div>
									<div v-if="scope.row.status == '3'">
										<span class='el-icon el-icon-status-alarm' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("GaoJingGuanLi") %></span>
									</div>
									<div v-if="scope.row.status == '4'">
										<span class='el-icon el-icon-status-upgrading' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("ShengJiZhong") %></span>
									</div>
									<div v-if="scope.row.status == '5'">
										<span class='el-icon el-icon-status-upgrade-success' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("ShengJi") %><%=rb.getString("ChengGong") %></span>
									</div>
								</template>
							</el-table-column>
							<el-table-column label="<%=rb.getString("XiaoZhanBianMa") %>" prop="serial_number" width="180"></el-table-column>
							<el-table-column label="<%=rb.getString("EUName") %>" prop="device_name"></el-table-column>
							<el-table-column label="<%=rb.getString("MoKuaiXingHao") %>" prop="model_name"></el-table-column>
							<el-table-column label="<%=rb.getString("RuanJianBanBen") %>" prop="software_version" width="140"></el-table-column>
							<el-table-column label="IP" prop="ip"></el-table-column>
						</el-ctable>
						<el-cmenu ref="menu_eu" :data="menus_eu" @click="clickMenuEu"></el-cmenu>
					</div>
				</div>
			</el-collapse-item>
			<el-collapse-item name="RU">
				<template slot='title'>
					<p style="display:inline-block;margin-left:40px;">
						<span class="title-icon" style="vertical-align:sub"></span>
						<span style="font-size:14px;font-weight:bold">RU <%=rb.getString("SheBeiLieBiao") %></span>
					</p>
				</template>
				<div :class="boardCardShow == true ? 'rightContentCls' : 'noCardTableContentCls' ">
					<div style='height:100%;margin:10px 0px;'>
						<el-ctable id="ruTable" ref="ruTable" :url="ruUrl"  :query-params="queryParams"
							:height="height" pagination="true" rownumber="true" style="margin-right:100px;border:1px solid #E9E9E9;">
							<el-table-column v-if="optBtnShow" label="" width="30" prop="" class-name="no-text-tips">
								<template slot-scope="scope">
									<div class="el-icon el-icon-operation-more" @click="optClickRu(scope.row,event)" v-clickoutside="handerClose" style="cursor: pointer;"></div>
								</template>
							</el-table-column>
							<el-table-column label="<%=rb.getString("LuYouSuoYin") %>" prop="route_index" sortable="true" width="140"></el-table-column>
							<el-table-column label="<%=rb.getString("BanKaID") %>" prop="card_id" v-if="boardCardShow"></el-table-column>
							<el-table-column label="<%=rb.getString("XuLieHao") %>" prop="index"></el-table-column>
							<el-table-column label="<%=rb.getString("ZhuangTai") %>" prop="status" width="100">
								<template slot-scope="scope">
									<div v-if="scope.row.status == '1'">
										<span class='el-icon el-icon-status-conn-on' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("ZhengChang") %></span>
									</div>
									<div v-if="scope.row.status == '2'">
										<span class='el-icon el-icon-status-conn-off' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("LiXian") %></span>
									</div>
									<div v-if="scope.row.status == '3'">
										<span class='el-icon el-icon-status-alarm' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("GaoJingGuanLi") %></span>
									</div>
									<div v-if="scope.row.status == '4'">
										<span class='el-icon el-icon-status-upgrading' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("ShengJiZhong") %></span>
									</div>
									<div v-if="scope.row.status == '5'">
										<span class='el-icon el-icon-status-upgrade-success' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("ShengJi") %><%=rb.getString("ChengGong") %></span>
									</div>
								</template>
							</el-table-column>
							<el-table-column label="<%=rb.getString("XiaoZhanBianMa") %>" prop="serial_number" width="180"></el-table-column>
							<el-table-column label="<%=rb.getString("RUName") %>" prop="device_name"></el-table-column>
							<el-table-column label="<%=rb.getString("ZuiDaFaSheGongLv") %>" prop="max_tx_power"></el-table-column>
							<el-table-column label="<%=rb.getString("FangSheZhuangTai") %>" prop="rf_tx_status">
								<template slot-scope="scope">
									<div v-if="scope.row.rf_tx_status == 'false' || scope.row.rf_tx_status == '0'">
										<span class='el-icon el-icon-status-disable' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("GuanBi") %></span>
									</div>
									<div v-if="scope.row.rf_tx_status == 'true' || scope.row.rf_tx_status == '1'">
										<span class='el-icon el-icon-status-enable' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("KaiQi") %></span>
									</div>
								</template>
							</el-table-column>
							<el-table-column label="<%=rb.getString("MoKuaiXingHao") %>" prop="model_name" width="140"></el-table-column>
							<el-table-column label="<%=rb.getString("RuanJianBanBen") %>" prop="software_version" width="140"></el-table-column>
							<el-table-column label="IP" prop="ip"></el-table-column>
						</el-ctable>
						<el-cmenu ref="menu_ru" :data="menus_ru" @click="clickMenuRu"></el-cmenu>
					</div>
				</div>
			</el-collapse-item>
		</el-collapse>
					
					
	</div>
	<el-dialog style='margin-top:20vh'  title='<%=rb.getString("SheZhi") %>' width='450px' :visible.sync='setVisible' :append-to-body="true" :close-on-click-modal="false" @close="closeSetPower">
		<div style='display:flex;align-items:baseline'>
			<span style="display: inline-block;min-width: 85px;"><%=rb.getString("MingCheng") %></span>
			<div style='display:inline-block;margin-left:20px;margin-right:20px;height:50px;'>
				<el-input v-model='nameVal' style="width:130px;" @blur="checkName"></el-input>
				<p v-show="showEUTip" style='font-size:12px;margin-top:5px;color:#FA5555'>{{tipNameMessage}}</p>
			</div>
			<span style="color:#999"><%=rb.getString("ZiFu") %>(0-48)</span>
		</div>
		<div style='display:flex;align-items:baseline'>
			<span style="display: inline-block;min-width: 85px;"><%=rb.getString("ZuiDaFaSheGongLv") %></span>
			<div style='display:inline-block;margin-left:20px;margin-right:20px;height:50px;'>
				<el-input v-model='powerVal' style="width:130px;" @blur="checkPower" maxlength="48"></el-input>
				<p v-show="showTip" style='font-size:12px;margin-top:5px;color:#FA5555'>{{tipMessage}}</p>
			</div>
			<span style="color:#999"><%=rb.getString("FanWei") %>(0-43)</span>
		</div>
		<div style='margin-top:45px;margin-left:180px;'>
			<el-button @click="savePower"  type="primary"><%=rb.getString("QueDing")%></el-button>
			<el-button @click="closeSetPower"><%=rb.getString("QuXiao")%></el-button>
		</div>
	</el-dialog>
	
	<el-dialog style='margin-top:20vh'  title='<%=rb.getString("SheZhi") %>' width='450px' :visible.sync='setEUVisible' :append-to-body="true" :close-on-click-modal="false" @close="closeSetName">
		<div style='display:flex;align-items:baseline'>
			<span><%=rb.getString("MingCheng") %></span>
			<div style='display:inline-block;margin-left:20px;margin-right:20px;height:50px;'>
				<el-input v-model='nameVal' style="width:180px;" @blur="checkName"></el-input>
				<p v-show="showEUTip" style='font-size:12px;margin-top:5px;color:#FA5555'>{{tipNameMessage}}</p>
			</div>
			<span style="color:#999"><%=rb.getString("ZiFu") %>(0-48)</span>
		</div>
		<div style='margin-top:45px;margin-left:180px;'>
			<el-button @click="saveName"  type="primary"><%=rb.getString("QueDing")%></el-button>
			<el-button @click="closeSetName"><%=rb.getString("QuXiao")%></el-button>
		</div>
	</el-dialog>

	<!--板卡修改弹窗-->
	<el-dialog title='<%=rb.getString("XiuGai") %>' id="editBoardCardDialog" :visible.sync="showBoardCardDialog" top="30vh" ref="editBoardCardDialog" 
		width="500px" :close-on-click-modal="false"  @close='closeBoardCardDialog' append-to-body>
		<el-form  :model="editBoardCardForm" ref="editBoardCardForm"  label-position="left" id="editBoardCardForm">
			<el-form-item  style="margin-left:40px;margin-bottom:0px;" label="Cell Count" prop='enb_id' label-width="120px">
				<el-radio-group v-model='editBoardCardForm.cellCount' style="margin-top:10px;">
					<el-radio :label="1">1</el-radio>
					<el-radio :label="2">2</el-radio>
					<el-radio :label="3">3</el-radio>
				</el-radio-group>
			</el-form-item>
			<el-form-item prop='cell_id' style="margin-left:40px;margin-bottom:0px;" label="Duplex Mode" label-width="120px">
				<el-radio-group v-model='editBoardCardForm.duplexMode' style="margin-top:10px;">
					<el-radio label="TDD">TDD</el-radio>
					<el-radio label="FDD">FDD</el-radio>
				</el-radio-group>
			</el-form-item>
		</el-form>
		<span slot="footer">
			<div>
				<el-button type="primary" @click="editBoardCardSubmit"><%=rb.getString("QueDing")%></el-button>
				<el-button @click="closeBoardCardDialog"><%=rb.getString("QuXiao")%></el-button>
			</div>
		</span>
	</el-dialog>
</div>
<script>
	var distributedVue = new Vue({
		el:'#distributeDetail',
		data(){
			return{
                enbSelectedRow:{},
				height:'100%',
				euUrl:'${ctx}/cell/nxp/queryEUInfos.action',
				ruUrl:'${ctx}/cell/nxp/queryRUInfos.action',
				queryParams:{
					smallCellCode : "${smallCellCode}"
				},
				queryParamsEUAndRU:{
					smallCellCode : "${smallCellCode}",
					searchText: ''
				},
				statisticData:[],
				statisticCount:{
					"euOnlineCount":"0",
					"ruOnlineCount":"0",
					"euTotalCount":"0",
					"ruTotalCount":"0"
				},
				menus_eu:[],
				menus_ru:[],
				setVisible:false,
				setEUVisible: false,
				showTip:false,
				showEUTip:false,
				powerVal:'',
				nameVal: '',
				rowDataEu:[],
				rowDataRu:[],
				operType:'',
				tipMessage:'',
				tipNameMessage:'',
				boardCardData:[{
					cardStatus:'1',
					cardId:'1_1',
					cellNum:'2',
					duplexMode:'FDD',
					cardSn:'11000'
				}],
				stationCollapse:['BC','Statistic','EU','RU'],
				boardCardUrl:'${ctx}/pm/nxp/getSlots.action',
				showBoardCardDialog:false,
				editBoardCardForm:{
					cellCount:'',
					duplexMode:''
				},
			}
		},
		computed: {
			boardCardShow() { // 当前设备数据
				return this.enbSelectedRow.product == 'PM-B4860' ? true : false;
			},
			optBtnShow() {
				return writableMap['CODE_ENB_MONITOR'] == true;
			},
		},
		
		methods:{
			// 初始化
			init(row,code,sn){
				var vm = this;
                vm.enbSelectedRow = row;
				vm.getEuAndRuData();

			},
            getEuAndRuData(){
                var vm = this,
                    params ={
                        "smallCellCode" : "${smallCellCode}",
                        'searchText': vm.queryParamsEUAndRU.searchText || '',
                        "page":"1",
                        "rows":"50",
                        "sort":"",
                        "order":""
                    };
				axios.post("${ctx}/cell/nxp/queryInfosByEuAndRu.action",stringify(params)).then(function(response){
					var data = response.data;
					
					if(data.rows){
						vm.statisticData = data.rows;
					}
					if(data.properties){
						vm.statisticCount = data.properties;
					}
				})
            },
			optClick(row,ev){
				var vm = this;
				vm.rowDataEu = row;
				vm.operType = 'eu';
				var connFlag = false;
				if (row.status != "2"){
					connFlag = true;
				}
				
				vm.menus_eu = [
					{label:'<%=rb.getString("ChongQi")%>',cls:'el-icon el-icon-operation-reboot',code:'reboot',disable:!connFlag},
					{label:'<%=rb.getString("SheZhi")%>',cls:'el-icon el-icon-operation-settings',code:'setting',disable:!connFlag},
					{label:'<%=rb.getString("ShanChu")%>',cls:'el-icon el-icon-operation-delete',code:'deleteEu',disable:connFlag}
				]
				vm.$nextTick(function(){
					document.body.click();
					vm.$refs.menu_eu.show(ev);
				})
			},
			optClickRu(row,ev){
				var vm = this;
				var product =  vm.enbSelectedRow.product;
				var showSetFlag = true;
				if(product == 'DXDF'){
					showSetFlag = false;
				}
				vm.rowDataRu = row;
				vm.operType = 'ru';
				var rfText = '';
				var rfIcon = '';
				if(row.rf_tx_status == 'false' || row.rf_tx_status == '0'){
					rfText = '<%=rb.getString("ShePinCaoZuo")%> <%=rb.getString("Kai")%>';
					rfIcon = 'el-icon el-icon-operation-enable1';
				}else{
					rfText = '<%=rb.getString("ShePinCaoZuo")%> <%=rb.getString("Guan")%>';
					rfIcon = 'el-icon el-icon-operation-disable1';
				}
				
				//设备不在线时，仅能操作删除 
				var connFlag = false;
				if (row.status != "2"){
					connFlag = true;
				}
				
				vm.menus_ru = [
					{label:'<%=rb.getString("ChongQi")%>',cls:'el-icon el-icon-operation-reboot',code:'reboot',disable:!connFlag},
					{label:rfText,cls:rfIcon,code:'rf',disable:!connFlag},
					{label:'<%=rb.getString("SheZhi")%>',cls:'el-icon el-icon-operation-settings',code:'setting',show:showSetFlag,disable:!connFlag},
					{label:'<%=rb.getString("ShanChu")%>',cls:'el-icon el-icon-operation-delete',code:'deleteRu',disable:connFlag}
				]
				vm.$nextTick(function(){
					document.body.click();
					vm.$refs.menu_ru.show(ev);
				})
			},
			handerClose(){
				this.$refs.menu_eu.hide();
				this.$refs.menu_ru.hide();
			},
			clickMenuEu(ev){
				var codes = {
						reboot:this.reboot,
						setting:this.setEu,
						deleteEu:this.deleteDevice
				}
				if(codes[ev.code]){
					codes[ev.code]();
				}
			},
			clickMenuRu(ev){
				var codes = {
						reboot:this.reboot,
						rf:this.changeRfRu,
						setting:this.setRu,
						deleteRu:this.deleteDevice
				}
				if(codes[ev.code]){
					codes[ev.code]();
				}
			},
			checkPower(){
				var reg=/^\d+$/;
				var val = this.powerVal;
				if(val == '' && (val !== 0 && val !== '0')){
					this.tipMessage = '<%=rb.getString("QingShuRu")%>'
					this.showTip = true;
				}else if(!reg.test(val) || val < 0 || val > 43){
					this.tipMessage = '<%=rb.getString("CuoWuGeShi")%>';
					this.showTip = true;
				}else{
					this.tipMessage = '';
					this.showTip = false;
				}
			},
			checkName() {
				var reg=/^[a-zA-Z]+$/;
				var val = this.nameVal;
				if(val && val.length>48){
					this.tipNameMessage = '<%=rb.getString("ZiFuFuShu")%><%=rb.getString("ZiFuChang")%>'+': (0-48)';
					this.showEUTip = true;
				}else{
					this.tipNameMessage = '';
					this.showEUTip = false;
				}
			},
			savePower(){
				var vm = this;
				vm.checkPower();
				vm.checkName();
				if(vm.showTip || vm.showEUTip){
				}else{
					var oldName = vm.rowDataRu.device_name || '',
						oldPower = vm.rowDataRu.max_tx_power||'',
						isNameChanged = oldName != vm.nameVal,
						isPowerChanged = oldPower != vm.powerVal,
						itemCodes = [];
					if(!(isNameChanged || isPowerChanged)) {
						vm.$message.warning('<%=rb.getString("WuCanShuBianHua")%>');
						return;
					}

					isNameChanged && itemCodes.push('deviceName');
					isPowerChanged && itemCodes.push('maxTxPower');
					
					var params = {
							item: itemCodes.join(','),
							deviceName: vm.nameVal,
							type: 'ru',
							index: vm.rowDataRu.index,
							euIndex: vm.rowDataRu.eu_index,
							smallCellCode: vm.rowDataRu.small_cell_code,
							serialNumber: vm.rowDataRu.serial_number,
							maxTxPower: vm.powerVal
					}
					axios({
						method:'post',
						url:'${ctx}/cell/nxp/basicSetting.action',
						headers:{
							'ContentType':'application/json'
						},
						data:params
					}).then(function(response){
						var data = response.data;
						if(data["success"]){
							vm.$refs.ruTable.refresh();
							vm.closeSetPower();
						}else{
							vm.$message.error(data["message"])
						}
					})
				}
			},
			saveName() {
				var vm = this;
				vm.checkName();
				if(vm.showEUTip){
				}else{
					var oldName = vm.rowDataEu.device_name || '';
					
					if(oldName == vm.nameVal) {
						vm.$message.warning('<%=rb.getString("WuCanShuBianHua")%>');
						return;
					}
					var params = {
							item: 'deviceName',
							deviceName: vm.nameVal,
							type: 'eu',
							index: vm.rowDataEu.index,
							smallCellCode: vm.rowDataEu.small_cell_code,
							serialNumber: vm.rowDataEu.serial_number
					}
					axios({
						method:'post',
						url:'${ctx}/cell/nxp/basicSetting.action',
						headers:{
							'ContentType':'application/json'
						},
						data:params
					}).then(function(response){
						var data = response.data;
						if(data["success"]){
							vm.$refs.euTable.refresh();
							vm.closeSetName();
						}else{
							vm.$message.error(data["message"])
						}
					})
				}
			},
			setRu(){
				var vm = this;
				vm.setVisible = true;
				vm.nameVal = vm.rowDataRu.device_name || '';
				vm.powerVal = (vm.rowDataRu.max_tx_power===undefined||vm.rowDataRu.max_tx_power===null)?'':vm.rowDataRu.max_tx_power;
			},
			setEu() {
				var vm = this;
				vm.setEUVisible = true;
				vm.nameVal = vm.rowDataEu.device_name || '';
			},
			closeSetPower(){
				this.nameVal = '';
				this.powerVal = '';
				this.showTip = false;
				this.setVisible = false;
			},
			closeSetName() {
				this.nameVal = '';
				this.showEUTip = false;
				this.setEUVisible = false;
			},
			deleteDevice(){
				var vm = this;
				var table,url,confirmStr;
				if(vm.operType == 'eu'){
					var params = {
							small_cell_code : vm.rowDataEu.small_cell_code,
							index : vm.rowDataEu.index,
							serial_number : vm.rowDataEu.serial_number,
					}
					table = vm.$refs.euTable;
					url = "${ctx}/cell/nxp/deleteEuInfo.action";
					confirmStr = '<%=rb.getString("QueDingShanChuEUTongShiShanChuRU")%>'
				}else{
					var params = {
							small_cell_code : vm.rowDataRu.small_cell_code,
							serial_number : vm.rowDataRu.serial_number,
					}
					table = vm.$refs.ruTable;
					url = "${ctx}/cell/nxp/deleteRuInfo.action";
					confirmStr = '<%=rb.getString("QueDingShanChuRU")%>';
				}
				
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					customClass:"warningConfirm",
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					axios({
						method:'post',
						url:url,
						headers:{
							'ContentType':'application/json'
						},
						data:stringify(params)
					}).then(function(response){
						var data = response.data;
						if(data["success"]){
							vm.$message({
								type:'success',
								message:'<%=rb.getString("ChengGong")%>'
							})
							table.refresh();
							
							if(vm.operType == 'eu'){
								vm.$refs.ruTable.refresh();
							}
						}else{
							vm.$message.error(data["message"])
						}
					})
				})
			},
			reboot(){
				var vm = this;
				var table;
				if(vm.operType == 'eu'){
					var params = {
							smallCellCode : vm.rowDataEu.small_cell_code,
							type : vm.operType,
							index : vm.rowDataEu.index,
							euIndex : '',
							serialNumber : vm.rowDataEu.serial_number,
							routeIndexx : vm.rowDataEu.route_index
					}
					table = vm.$refs.euTable;
				}else{
					var params = {
							smallCellCode : vm.rowDataRu.small_cell_code,
							type : 'ru',
							index : vm.rowDataRu.index,
							euIndex : vm.rowDataRu.eu_index,
							serialNumber : vm.rowDataRu.serial_number,
							routeIndex : vm.rowDataRu.route_index
					}
					table = vm.$refs.ruTable;
				}
				var confirmStr = '<%=rb.getString("QueDingChongQiSheBei")%>'
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					customClass:"warningConfirm",
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					axios({
						method:'post',
						url:'${ctx}/cell/nxp/reboot.action',
						headers:{
							'ContentType':'application/json'
						},
						data:params
					}).then(function(response){
						var data = response.data;
						if(data["success"]){
							vm.$message({
								type:'success',
								message:'<%=rb.getString("ChengGong")%>'
							})
							table.refresh();
						}else{
							vm.$message.error(data["message"])
						}
					})
				})
			},
			changeRfRu(){
				var vm = this;
				var product =  vm.enbSelectedRow.product;
				var params = {
						item: 'rfTxStatus',
						type:'ru',
						index : vm.rowDataRu.index,
						euIndex : vm.rowDataRu.eu_index,
						smallCellCode : vm.rowDataRu.small_cell_code,
						serialNumber : vm.rowDataRu.serial_number
				}
				if(product == 'DXDF'){
					if(vm.rowDataRu.rf_tx_status == '0'){
						params.rfTxStatus = '1'
					}else{
						params.rfTxStatus = '0'
					}
				}else{
					params.rfTxStatus = vm.rowDataRu.rf_tx_status=='true'?false:true;
				}
				axios({
					method:'post',
					url:'${ctx}/cell/nxp/basicSetting.action',
					headers:{
						'ContentType':'application/json'
					},
					data:params
				}).then(function(response){
					var data = response.data;
					if(data["success"]){
						vm.$refs.ruTable.refresh();
					}else{
						vm.$message.error(data["message"])
					}
				})
			},
			// 打开板卡修改弹窗
			editBoardCard(row){
				var vm = this;
				vm.boardCardRow = row;
				vm.editBoardCardForm.cellCount = row.cellNum;
				vm.editBoardCardForm.duplexMode = row.duplexMode;
				vm.showBoardCardDialog = true;
			},
			// 重启板卡
			rebootBoardCard(row){
				var vm = this,
					url = '${ctx}/pm/nxp/slotReboot.action',
					params = {
						smallCellCode:'${smallCellCode}',
						cardId:row.cardId
					};
				var confirmStr = '<%=rb.getString("QueDingChongQiSheBei")%>'
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					customClass:"warningConfirm",
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					axios.post(url,stringify(params)).then(function(response){
						let data = response.data
						if(data["success"]){
							vm.$message({
								type:'success',
								message:'<%=rb.getString("ChengGong")%>'
							})
							vm.$refs.boardCardTable.refresh();
						}else{
							vm.$message.error(data["message"])
						}
					}).catch(function(error){})
				})

			},
			// 修改板卡提交
			editBoardCardSubmit(){
				var vm = this,
					url = '${ctx}/pm/nxp/updateSlotInfo.action',
					params = {
						smallCellCode:'${smallCellCode}',
						cardId:vm.boardCardRow.cardId,
						duplexMode:vm.editBoardCardForm.duplexMode,
						cellNum:vm.editBoardCardForm.cellCount
					};
				axios.post(url,stringify(params)).then(function(response){
						let data = response.data
						if(data["success"]){
							vm.$message({
								type:'success',
								message:'<%=rb.getString("ChengGong")%>'
							})
							vm.$refs.boardCardTable.refresh();
							vm.closeBoardCardDialog();
						}else{
							vm.$message.error(data["message"])
						}
					}).catch(function(error){})
			},
			// 关闭弹窗
			closeBoardCardDialog(){
				var vm = this,
					params = {
						cellCount:'',
						duplexMode:''
					};
				vm.showBoardCardDialog = false;
				Object.assign(vm.editBoardCardForm,params)
			},
			//status EU,RU 列表查询
			euAndRuQuery(val) {
	       		var vm = this;
                vm.queryParamsEUAndRU.searchText = val;
	       		vm.getEuAndRuData();
	     	},
		},
        mounted(){
			eventBus.$off("enb-data").$on("enb-data",this.init)
		},
	})
</script>