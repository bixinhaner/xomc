<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<!-- 显示隐藏列 -->
<!-- <div class="showHideItem" id="showGsmOrHideItem"></div> -->
<div v-if="isExisted && isAdmin" class="fixed-right-msg" style="right: 60px;">
	<div v-if="msgExtend" style="margin-right: 20px;line-height:26px;">
		<span style="padding: 0px 10px;">{{collectSn}}</span>
		<span style="padding: 0px 5px;" v-if="taskTime==''">
			<i class="el-icon el-icon-status-yes" style="font-size: 12px;"></i>
			<%=rb.getString("ChengGong")%>
		</span>
		<span v-if="taskTime!=''" style="display: inline-block;padding: 2px 30px;background: #4d84ff;border-radius: 2px;margin: 0px 5px 2px 5px;"></span>
		<span v-if="taskTime!=''" style="border: 1px solid #e3e3e3;border-radius: 3px;padding: 2px 4px;">
			<span style="cursor: pointer;" @click="stopCollect">
				<i style="padding: 4px;background: red;height: 0px;display: inline-block;border-radius: 3px;"></i>
				<%=rb.getString("TingZhi")%>
			</span>
			<span style="margin-left: 5px;">
				{{taskTime}}
			</span>
		</span>
		<span style="margin-left: 20px;">
			<a class="collect-bt" @click="viewMsg"><%=rb.getString("ChaKan")%></a>
			<a class="collect-bt" @click="downloadMsg"><%=rb.getString("XiaZai")%></a>
			<a class="collect-bt" @click="clearMsg"><%=rb.getString("QingChu")%></a>
		</span>
	</div>
	<div @click="msgExtend = !msgExtend" class="foldBtnCls">
		<span v-if="msgExtend" class="el-icon el-icon-common-query-up"></span>
		<span v-if="!msgExtend" class="el-icon el-icon-common-query-down"></span>
	</div>
</div>
<!-- progress -->
<div v-if="!progressHide" class="fixed-right-msg" :style="{right: msgExtend?(taskTime==''?'550px':'650px'):'100px', display: 'flex', 'align-items': 'baseline', padding: '5px 0 5px 15px'}">
	<%=rb.getString("DaoChuJinDu")%>: <el-progress :percentage="percentage" style="width:120px;margin-left: 5px;" color="#f56c6c"></el-progress>
</div>
<div id="gsmTableHeadQuery" class="tableHeadQueryBoxCls" style="padding-right:120px;">
	<div class="headQueryBox">
		<div class="queryGroup">
			<el-input v-model="search_text" @keyup.enter.native="query" :placeholder='placeholderText' style="width:360px;"></el-input>
			<i @click='query' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
		</div>
	</div>
	<div v-for="(item,index) in advancedQueryItemList">
		<div v-if="item.type == 'checkbox' && item.isShow" style="margin-right:10px;">
			<el-popfilter
				:label='item.label'
				v-model="item.checkedItemList"
				:list="item.options"
				:visible.sync="item.isShow"
				:closable="true"
				@check-change="advanceQuery(item.type,item.value,item.checkedItemList)"
				@close="checkItemDel(item)">
			</el-popfilter>
		</div>
		<div v-if="item.type == 'select' && item.isShow" style="margin-right:10px;">
			<el-popfilter
				type="single"
				:label='item.label'
				v-model="item.selectVal"
				:list="item.options"
				:visible.sync="item.isShow"
				:closable="true"
				@check-change="advanceQuery(item.type,item.value,item.selectVal)"
				@close="checkItemDel(item)">
			</el-popfilter>
		</div>
		<div v-if="item.type == 'filter'" style="margin-right:10px;">
			<div class="advancedQueryItemBox" style="background: #FFF;">
				<el-popover :ref="'popover-'+item.value" trigger="click" placement="bottom-start"  @show="checkPopoverShow(item)" @hide="checkPopoverHide(item,'')">
					<div class="checkPopoverBoxCls">
						<el-checkbox-group v-model="item.checkedItemList">
							<el-checkbox v-for=" items in item.options" :key="items.value" :label="items.value">{{items.label}}</el-checkbox>
						</el-checkbox-group>
						<div class="buttonGroup">
							<el-button size="mini" type="primary" @click="checkPopoverSubmit(item)"><%=rb.getString("QueDing")%></el-button>
							<el-button size="mini" @click="checkPopoverHide(item,'del')"><%=rb.getString("QuXiao")%></el-button>
						</div>	
					</div>
					<div slot="reference" class="ItemAndIconBoxCls">
						<i class="el-icon el-icon-filterAdd"></i>
						{{item.label}}
					</div>
				</el-popover>
			</div>
		</div>
	</div>
	<div class="advancedQueryItemBox"  style="background: #FFF;" @click="clearFilterClick">
		<%=rb.getString("QingKongShaiXuan")%>
	</div>
	<!-- <div class="advancedQueryItemBox" style="background: #FFF;position:absolute;right:10px;top:7px;" @click="timeLockChange">
		<i :class="{'el-icon':true, 'placeholder-bt':true, 'placeholder-bt':true, 'last-bt':true, 'el-icon-monitor-lock':timeLocked, 'el-icon-monitor-refresh':!timeLocked}" 
			style="cursor:pointer;position:relative;margin-right:5px;"></i>
		<span v-show="timeLocked"><%=rb.getString("SuoDingShuaXin")%></span>
		<span v-show="!timeLocked"><%=rb.getString("DingShiShuaXin")%></span>
	</div> -->
	<el-dialog :visible.sync="collectInfoShow" top="10vh">
        <div slot="title">
            <span class="el-dialog__title"><%=rb.getString("XinXi")%></span>
            <i class="el-icon el-icon-operation-export" style="position: absolute;right: 45px;top: 9px;" @click="downloadMsg"></i>
        </div>
        <el-input v-model="collectContent" type="textarea" rows="25" readonly class="none-border"></el-input>
    </el-dialog>
</div>
<script>
	var queryGsmVue = new Vue({
			el:'#gsmtoolbar_tableHomeCellList',
			data(){
				return{
					placeholderText:'<%=rb.getString("BSCBianMa")%>/ <%=rb.getString("BSCMingCheng")%>/ <%=rb.getString("IPDiZhi")%>/ MAC/ <%=rb.getString("SuoShuBSCBianMa")%>',
					advancedQueryItemList:[
						{
							type:'checkbox',
							isShow:true,
							isIndeterminate:false,
							checkAll:false,
							popoverShow:false,
							checkedItemList:[],
							oldCheckedItemList:[],
							label:'<%=rb.getString("ZaiXianZhuangTai") %>',
							options:[
								{label:'<%=rb.getString("LianJieZhengChang")%>',value:"1"},
								{label:'<%=rb.getString("LianJieDuanKai")%>',value:"0"},
								{label:'<%=rb.getString("TongBuZhong")%>',value:"3"},
								{label:'<%=rb.getString("TongBuShiBai")%>',value:"2"}
							],
							value:'connection_status',
						},
                        {
							type:'select',
							isShow:true,
							popoverShow:false,
							selectVal:'',
							label:'<%=rb.getString("ShiFouJiHuo") %>',
							options:[
								{label:'<%=rb.getString("QuanBu")%>',value:''},
								{label:'<%=rb.getString("JiHuo")%>',value:"1"},
								{label:'<%=rb.getString("QuJiHuo")%>',value:"0"}
							],
							value:'op_state',
						},
                        {
							type:'checkbox',
							isShow:true,
							isIndeterminate:false,
							checkAll:false,
							popoverShow:false,
							checkedItemList:[],
							oldCheckedItemList:[],
							label:'<%=rb.getString("ChanPinLeiXingBiaoZhi") %>',
							options:[
                                {label:'BSC',value:"BSC"},
                                {label:'BTS',value:"BTS"}
                            ],
							value:'product_model',
						},
						{
							type:'checkbox',
							isShow:false,
							isIndeterminate:false,
							checkAll:false,
							popoverShow:false,
							checkedItemList:[],
							oldCheckedItemList:[],
							label:'<%=rb.getString("SheBeiXingHaoMing") %>',
							options:[],
							value:'model_name',
						},
						{
							type:'checkbox',
							isShow:false,
							isIndeterminate:false,
							checkAll:false,
							popoverShow:false,
							checkedItemList:[],
							oldCheckedItemList:[],
							label:'<%=rb.getString("SoftwareVersion") %>',
							options:[],
							value:'software_version',
						},
						{
							type:'checkbox',
							isShow:false,
							isIndeterminate:false,
							checkAll:false,
							popoverShow:false,
							checkedItemList:[],
							oldCheckedItemList:[],
							label:'<%=rb.getString("FirmwareVersion") %>',
							options:[],
							value:'firmware_version',
						},
						{
							type:'checkbox',
							isShow:false,
							isIndeterminate:false,
							checkAll:false,
							popoverShow:false,
							checkedItemList:[],
							oldCheckedItemList:[],
							label:'<%=rb.getString("SheBeiZu") %>',
							options:[],
							value:'group_id',
						},
                        {
							type:'checkbox',
							isShow:false,
							isIndeterminate:false,
							checkAll:false,
							popoverShow:false,
							checkedItemList:[],
							oldCheckedItemList:[],
							label:'<%=rb.getString("SuoShuBSCBianMa") %>',
							options:[],
							value:'bscSerialnumber',
						},
						{
							type:'filter',
							popoverShow:false,
							checkedItemList:['connection_status','op_state','product_model'],
							oldCheckedItemList:['connection_status','op_state','product_model'],
							label:'<%=rb.getString("TianJiaShuaiXuan") %>',
							options:[
								{label:'<%=rb.getString("ZaiXianZhuangTai") %>',value:"connection_status"},
                                {label:'<%=rb.getString("ShiFouJiHuo")%>',value:"op_state"},
								{label:'<%=rb.getString("ChanPinLeiXingBiaoZhi")%>',value:"product_model"},
								{label:'<%=rb.getString("SheBeiXingHaoMing")%>',value:"model_name"},
								{label:'<%=rb.getString("SoftwareVersion")%>',value:"software_version"},
								{label:'<%=rb.getString("FirmwareVersion")%>',value:"firmware_version"},
								{label:'<%=rb.getString("SheBeiZu")%>',value:"group_id"},
                                {label:'<%=rb.getString("SuoShuBSCBianMa")%>',value:"bscSerialnumber"},
							],
							value:'add_filter',
						},
					],
					search_text:'',
					
					queryParams:{
						TimeZone : timeZone
					},
					timeLocked: true,

                    isExisted: false,
					collectDeviceCode: '',
					collectSn: '',
					taskTime: '',
					msgExtend: false,
					collectInfoShow: false,
					collectContent: '',
					percentage: ''
				}
			},
			computed: {
				isAdmin() {
					return is_super_user == 'true'
				},
				progressHide() {
					return ['',null,undefined].includes(this.percentage);
				}
			},
			methods:{
				queryProgress() {
					var vm = this,
						url = '${ctx}/cell/cpeinfos/getExportGSMCellsProgress.action';

					clearInterval(gsmProgressInterval);
					gsmProgressInterval = setInterval(function() {
						axios.post(url).then(function(res){
							var data = res.data;

							vm.percentage = data - 0;

							if(vm.percentage == 100 || data === '') {
								clearInterval(gsmProgressInterval);
								setTimeout(function(){
									vm.percentage = '';
								},1500)
							}
						});
					},1000);
				},
				timeLockChange() {
					var vm = this;
            		
					vm.timeLocked = !vm.timeLocked;
				},
				// 模糊搜索
				query(){
					var vm = this;
					vm.queryParams.search_text = this.search_text;
					vm.queryParams['like_fields'] = 'serial_number,host_name,cell_ip,mac_address,cell_identity,phycellid';
					Object.assign(gsmvm.queryParams, vm.queryParams);
				},
				// 高级查询 确定事件
				advanceQuery(type,paramsItem,value){
					var vm = this,
						params ={};
					if(type == 'select'){
						params[paramsItem] = value;
					}else{
                        if(paramsItem == 'product_model'){
                            if(value.length == 0){
                                params[paramsItem] = 'BSC,BTS';
                            }else{
                                params[paramsItem] = value.join(',');
                            }
                        }else{
                            params[paramsItem] = value.join(',');
                        }
					}
                    if(params.op_state){
                        vm.advancedQueryItemList.map((items)=>{
                            if('product_model' == items.value && items.isShow){
                                items.checkedItemList = ['BTS'];
                                items.oldCheckedItemList = ['BTS'];
                            }
                        })
                        params.product_model = 'BTS';
                    }
                    if(params.product_model && (params.product_model.includes('BSC'))){
                        vm.advancedQueryItemList.map((items)=>{
                            if('op_state' == items.value && items.isShow){
                                items.selectVal = '';
                            }
                        })
                        params.op_state = '';
                    }
					Object.assign(gsmvm.queryParams, params);
				},
				// 高级搜索下拉弹窗 展开事件
				checkPopoverShow(item){
					var vm = this;
					vm.advancedQueryItemList.map((items)=>{
						if(item.value == items.value){
							items.popoverShow = true;
						}
					})
				},
				// 高级搜索下拉弹窗 收起事件
				checkPopoverHide(item,types){
					var vm = this,
						checkCount = 0,
						allCount = item.options.length;
                    checkCount = item.oldCheckedItemList.length
					vm.advancedQueryItemList.map((items)=>{
						if(item.value == items.value){
							items.popoverShow = false;
							items.checkedItemList = items.oldCheckedItemList;
						}
					})
					if(types == 'del'){
						document.body.click();
					}
				},
				// 高级搜索下拉弹窗 选择提交
				checkPopoverSubmit(item){
					var vm = this,
						params ={};
                    vm.advancedQueryItemList.map((items)=>{
                        if(item.value == items.value){
                            items.oldCheckedItemList = items.checkedItemList;
                        }else{
                            var result = item.checkedItemList.includes(items.value);
                            if(!result){
                                items.isShow = false;
                                if(items.value == 'product_model'){
                                    params[items.value] = 'BSC,BTS';
                                }else{
                                    params[items.value] = '';
                                }
                                if(items.type != 'select'){
                                    items.checkedItemList = [];
                                    items.oldCheckedItemList = [];
                                    items.checkAll = false;
                                    items.isIndeterminate = false;
                                }else{
                                    items.selectVal = '';
                                }
                            }else{
                                items.isShow = true;
                            }
                        }
                    })
                    if(params.op_state == ''){
                        vm.advancedQueryItemList.map((items)=>{
                            if('product_model' == items.value){
                                if(items.isShow == false){
                                    params.product_model = 'BSC,BTS';
                                };
                            }
                        })
                    }
					Object.assign(gsmvm.queryParams, params);
					document.body.click();
				},
				// 筛选项删除事件
				checkItemDel(item){
					var vm = this,
						params ={};
					vm.advancedQueryItemList.map((items)=>{
						if(item.value == items.value){
							items.isShow = false;
							if(items.type != 'select'){
								items.checkedItemList = [];
								items.oldCheckedItemList = [];
								items.checkAll = false;
								items.isIndeterminate = false;
							}else{
								items.selectVal = '';
							}
                            if(items.value == 'product_model'){
                                params[items.value] = 'BSC,BTS';
                            }else{
                                params[items.value] = '';
                            }
						}
						if(items.value == 'add_filter'){
							var idx = items.checkedItemList.indexOf(item.value),
								oldIdx = items.oldCheckedItemList.indexOf(item.value);
							if(idx>-1){
								items.checkedItemList.splice(idx,1)
							}
							if(oldIdx>-1){
								items.oldCheckedItemList.splice(idx,1)
							}
						}
					})
                    if(params.op_state == ''){
                        vm.advancedQueryItemList.map((items)=>{
                            if('product_model' == items.value){
                                if(items.isShow == false){
                                    params.product_model = 'BSC,BTS';
                                };
                            }
                        })
                    }
					Object.assign(gsmvm.queryParams, params);
				},
				// 清除筛选
				clearFilterClick(){
					var vm = this,
						params = {
							halodSerialNumbers:''
						};
					vm.advancedQueryItemList.map((items)=>{
						if(items.isShow && items.isShow== true ){
							if(items.type == 'checkbox'){
								items.checkedItemList = [];
								items.oldCheckedItemList = [];
								items.checkAll = false;
								items.isIndeterminate = false;
							}else if(items.type == 'select'){
								items.selectVal = '';
							}
							if(items.type == 'checkbox' || items.type == 'select'){
                                if(items.value == 'product_model'){
                                    params[items.value] = 'BSC,BTS';
                                }else{
                                    params[items.value] = '';
                                }
							}
						}
					})
					Object.assign(gsmvm.queryParams, params);
					document.body.click();
				},
				// 初始化
				init(){
					var vm = this;
					axios.post("${ctx}/cell/cpeinfos/getCellVersionList.action?isGSM=1").then(function(response){
						var data = response.data ? response.data : [];
						var arr = [];
						data.map(function(item){
							arr.push({label:item.software_version,value:item.software_version})
						})
						vm.advancedQueryItemList.map((items)=>{
							if('software_version' == items.value){
								items.options = arr
							}
						})
					})
					axios.post("${ctx}/cell/cpeinfos/getFirmwareVersionList.action?isGnb=0&isGSM=1").then(function(response){
						var data = response.data ? response.data : [];
						var arr = [];
						data.map(function(item){
							arr.push({label:item.firmware_version,value:item.firmware_version})
						})
						vm.advancedQueryItemList.map((items)=>{
							if('firmware_version' == items.value){
								items.options = arr
							}
						})
					})
					axios.post("${ctx}/cell/cpeinfos/getModelNameList.action?isGnb=0&isGSM=1").then(function(response){
						var data = response.data ? response.data : [];
						var arr = [];
						data.map(function(item){
							arr.push({label:item.module_type,value:item.module_type})
						})
						vm.advancedQueryItemList.map((items)=>{
							if('model_name' == items.value){
								items.options = arr
							}
						})
					})
					axios.post("${ctx}/cell/cpeinfos/getDeviceGroupListByCell.action").then(function(response){
						var data = response.data ? response.data : [];
						var arr = [];
						data.map(function(item){
							if (item){
								arr.push({label:item.group_name,value:item.id})
							}
						})
						vm.advancedQueryItemList.map((items)=>{
							if('group_id' == items.value){
								items.options = arr;
							}
						})
					})
                    axios.post("${ctx}/cell/cpeinfos/getBSCSnForBTSList.action?isGSM=1").then(function(response){
						var data = response.data ? response.data : [];
						var arr = [];
						data.map(function(item){
							arr.push({label:item.bscSerialnumber,value:item.bscSerialnumber})
						})
						vm.advancedQueryItemList.map((items)=>{
							if('bscSerialnumber' == items.value){
								items.options = arr
							}
						})
					})
				},
				columnSetting() {
					$("#showGsmOrHideItem").slideDown();
				},
				reseteNBQuery(){
					var vm = this;
					vm.clearFilterClick();
					gsmvm.queryParams.halodSerialNumbers = '';
				},
                queryLatestInfo() {
					var vm = this,
                        params = {
                            type: 'enb',
                            operatorCode: operatorCodeGloab
                        };

                    axios.post('${ctx}/trace/queryLatestMessageTraceDeviceInfo.action', stringify(params)).then(function(res){
                        var data = res.data;

						if(data) {
                        	vm.collectSn = data.serialNumber;
							vm.collectDeviceCode = data.deviceCode;

							if(data.status == '0' && data.remainTime) {
								vm.startInterval(data.remainTime);
							}else if(data.status == '1'){
								vm.taskTime = '';
								vm.startInterval('00:01');
							}

							if(['',undefined].includes(data.status) && ['',undefined].includes(data.serialNumber)) {
								vm.isExisted = false;
							}else {
								vm.isExisted = true;
							}
						}
                    });
				},
                viewMsg() {
					var vm = this,
                        params = {
                            serialNumber: vm.collectSn,
                            type: 'enb'
                        };
                           
                    vm.collectInfoShow = true;

                    axios.post('${ctx}/trace/queryMessageTraceInfo.action', stringify(params)).then(function(res){
                        var data = res.data;

                        if(data) {
							vm.collectContent = data;
                    		vm.collectInfoShow = true;
                        }
                    });
                },
                downloadMsg() {
                    var vm = this,
                        params = {
                            serialNumber: vm.collectSn,
                            type: 'enb',
                            timeZone: timeZone
                        };

                    exportByForm('${ctx}/trace/downLoadMessageTraceInfo.action',params);
                },
				clearMsg() {
					var vm = this,
                        params = {
                            deviceCode: vm.collectDeviceCode,
                            serialNumber: vm.collectSn,
                            type: 'enb',
                            operatorCode: operatorCodeGloab
                        };

                    axios.post('${ctx}/trace/clear.action', stringify(params)).then(function(res){
                        var data = res.data;

                        if(data.success == true) {
                            vm.$message.success('<%=rb.getString("ChengGong")%>');
							vm.queryLatestInfo();
                        }else {
                            vm.$message.error(data.message);
                        }
                    });
				},
				stopCollect() {
					var vm = this,
                        params = {
                            deviceCode: vm.collectDeviceCode,
                            serialNumber: vm.collectSn,
                            type: 'enb',
                            operatorCode: operatorCodeGloab
                        };

                    axios.post('${ctx}/trace/stop.action', stringify(params)).then(function(res){
                        var data = res.data;

                        if(data.success == true) {
                            vm.$message.success('<%=rb.getString("ChengGong")%>');
							vm.queryLatestInfo();
                        }else {
                            vm.$message.error(data.message);
                        }
                    });
				},
				startInterval(time) {
					var vm = this,
						list = time.split(':'),
						totalTime = list[0]*60 + list[1]*1;

					clearInterval(gsmCollectInterval);
					
					gsmCollectInterval = setInterval(function() {
						totalTime -= 1;
						vm.taskTime = vm.formaterTime(Math.floor(totalTime/60)) + ':' + vm.formaterTime(totalTime%60);

						if(totalTime<1) {
							clearInterval(gsmCollectInterval);
							vm.taskTime = '';
						}
					},1000);
				},
				formaterTime(num) {

					return num < 10? '0'+num : num;
				},
			},
			mounted(){
				this.init();
                this.queryLatestInfo();
				setTimeout(closeLoading, 500);
			}
		})
</script>
