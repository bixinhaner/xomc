<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
#addGsmUeTracePage{
    padding: 10px 0px;
    box-sizing: border-box;
}
#addGsmUeTracePage .alarmBottomLine{
	background-color:#E9E9E9;
	width: 100%;
	height: 1px;
	margin-bottom: 30px; 
}
#addGsmUeTracePage .titleStyML{
	margin-left: 40px;
    margin-bottom: 15px;
}
#addGsmUeTracePage .el-form-item{
	margin: 0px 0px 10px 65px;
}
#addGsmUeTracePage .el-form-item .el-input{
    padding-top: 5px;
}
#addGsmUeTracePage .el-radio.is-bordered.el-radio--small{
	padding: 8px 15px 5px 10px;
}
#addGsmUeTracePage .validate-item .el-input-group__append{
    border:none;
    background:none;
}
#addGsmUeTracePage .validate-item .el-form-item__error{
    display:none;
}
#addGsmUeTracePage .validate-item .el-input__inner{
    width: 330px;
}
#addGsmUeTracePage .el-form-item__error{
    white-space: nowrap;
}
#addGsmUeTracePage .is-error .el-input-group__append, #addGsmUeTracePage .is-error .item-tip{
    color:#FA5555;
}
#addGsmUeTracePage .gsmUeTraceSelectedBoxCls{
    min-height: 50px;
    margin-left: 65px;
    border: 1px solid #E9E9E9;
    width: 80%;
    padding: 10px;
    box-sizing: border-box;
    margin-bottom: 20px;
}
#addGsmUeTracePage .gsmUeTraceSelectedBoxCls .el-tag--info{
    margin: 0px 10px 10px 0px;
}
</style>

<div id="addGsmUeTracePage" style="padding:10px 0px;">
	<el-form :model='ruleForm' :rules="rules" ref="ruleForm" label-position="left" label-width="140px" :disabled="optType == 'view'">
        <el-form-item label="Enable" prop="enable" style="margin-left: 45px;">
            <el-switch v-model="ruleForm.enable" active-value="1" inactive-value="0" active-color="#4D84FF" inactive-color="#BDC1C6" style="padding-top: 10px;"></el-switch>
        </el-form-item>
		<div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text commonColor"><%=rb.getString("JiBenXinXi")%></span>
		</div>
        <el-form-item label='IMSI' prop='imsi' class='validate-item'>
            <el-input maxlength=100  v-model="ruleForm.imsi" style='width: 330px;'>
                <template slot="append">Length: 15~16</template>
            </el-input>
        </el-form-item>
        <el-form-item label='<%=rb.getString("GenZongCanKaoHao")%>' prop='traceId' class='validate-item'>
            <el-input v-model="ruleForm.traceId" size="mini" style='width: 330px;' disabled></el-input>
        </el-form-item>
        <!-- <el-form-item label='<%=rb.getString("IPDiZhi")%>'>
            <el-form-item prop='ip' style="display: inline-block;margin-left: 0px;">
                <el-input v-model="ruleForm.ip" style='width: 200px;'></el-input>
            </el-form-item>
            <span style="padding: 0px 10px;">:</span>
            <el-form-item prop='port' style="display: inline-block;margin-left: 0px;">
                <el-input v-model="ruleForm.port" style='width: 100px;'></el-input>
            </el-form-item>
        </el-form-item> -->
        <el-form-item label='<%=rb.getString("JieKouLeiXing") %>' prop="NEInterface" style="margin-bottom: 20px;">
            <el-checkbox-group v-model="ruleForm.NEInterface" size="small">
                <el-checkbox label="1" border>A-Interface</el-checkbox>
                <el-checkbox label="2" border style='margin-left: 20px;'>Abits-Interface</el-checkbox>
            </el-checkbox-group>
        </el-form-item>
		<div class="alarmBottomLine"></div>
        <div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text commonColor"><%=rb.getString("SheBeiXuanZe")%></span>
            <span style="color: rgba(0, 0, 0, 0.32);">( <%=rb.getString("ZuiDuoXuanZe")%> 10 <%=rb.getString("ZuiDuoXuanZeDevice")%> )</span>
		</div>
        <div v-if="optType == 'add'">
            <el-ctable 
                id="selectDeviceList" 
                row-key="small_cell_code" 
                ref="deviceCtable" 
                :url='deviceListUrl'
                @selection-change='batchSelect'
                :height="height" 
                pagination="true" 
                :query-params="deviceParams"
                :limit="10"
                style='margin-left: 65px;border: 1px solid #E9E9E9;width: 80%;'
            >
                <template slot="toolbar">
                    <el-query type="normal" @query="queryDevice" placeholder="<%=rb.getString("XiaoZhanBianMa")%> / <%=rb.getString("BSCMingCheng")%>" style='margin-right: 10px;'></el-query>
                </template>
                <el-table-column type="selection" width="45" :reserve-selection="true"></el-table-column>
                <el-table-column prop="connection_status" width="50">
                    <template slot-scope="scope">
                        <div :class="{
                            'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
                        '':scope.row.have_connected==2,
                        'conn_exc':scope.row.connection_status=='Exception',
                        'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
                    </template>
                </el-table-column>
                <el-table-column prop='serial_number' label='<%=rb.getString("XiaoZhanBianMa")%>'></el-table-column>
                <el-table-column prop='host_name' label='<%=rb.getString("BSCMingCheng")%>'></el-table-column>
                <el-table-column prop="product_name" label='<%=rb.getString("ChanPinMingCheng")%>'></el-table-column>
            </el-ctable>
            <el-form-item prop='deviceCodes' label-width="0" style="margin-bottom: 20px;">
                <el-input v-model='ruleForm.deviceCodes' v-show="false"></el-input>
            </el-form-item>
        </div>
        <div v-if="optType == 'view'" class="gsmUeTraceSelectedBoxCls"> 
            <el-tag v-for="(item,index) in detailSelectedData" :key="index" type="info">{{item}}</el-tag>
        </div>
	</el-form>
	
</div>

<script type="text/javascript">
var addGsmUeTraceVue = new Vue({
	el:'#addGsmUeTracePage',
	data(){
		var vm = this,
        validateImsi = (rule,value,callback) => {
			var reg = /^[0-9]\d{14,15}$/
			if(value === '' || value === null || value === undefined){
				callback(new Error('Length: 15~16'))
			}else if(reg.test(value)){
				callback();
			}else{
				callback(new Error('Length: 15~16'))
			}
		},
		validateIP = (rule, value, callback) => {
            var reg = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/;
            
            var reg = /^[0-9]+$/;
			if(value === '' || value === null || value === undefined){
				callback(new Error('<%=rb.getString("QingShuRuZhengQueDeIpDiZhi")%>'))
			}else{
                if(reg.test(value)){
                    callback();
                }else{
                    callback(new Error("<%=rb.getString("QingShuRuZhengQueDeIpDiZhi")%>"));
                }
            }
        },
        validatePort = (rule, value, callback) => {
            var regNum = /^\d+$/;
            if (value === '' || value === null || value === undefined) {
                callback(new Error("<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%> 0~65535 <%=rb.getString("ZhengXing")%>"));
            } else {
                if (!regNum.test(value) || (value < 0 || value > 65535)) {
                    callback(new Error("<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%> 0~65535 <%=rb.getString("ZhengXing")%>"));
                } else {
                    callback();
                }
            }
        },
        validateDevice = (rule,value,callback) => {
			if(value === '' || value === null || value === undefined){
            	callback('<%=rb.getString("QingXuanZeGenZongSheBei") %>');
            }else{
            	callback();
            }
		},
        validateNEInterface = (rule,value,callback) => {
			if(vm.ruleForm.NEInterface.length == 0) {
            	callback('<%=rb.getString("BiTian") %>');
            }else{
            	callback();
            }
		};
		return {
			ruleForm:{
                enable:'1',
                imsi:'',
				traceId: '${traceId}',
                //ip:'',
                //port:'',
				NEInterface: [],
                deviceCodes:''
			},
			rules:{
                imsi: [{validator: validateImsi,trigger:'blur'}],
                //ip: [{validator: validateIP,trigger:'blur'}],
                //port: [{validator: validatePort,trigger:'blur'}],
                NEInterface: [{validator: validateNEInterface,trigger:'change'}],
                deviceCodes: [{validator: validateDevice,trigger:'blur'}],
                
			},
            height:'270px',
            deviceListUrl:'${ctx}/bsc/signaling/getBSCDeviceInfoList.action',
            deviceParams:{
				timeZone:timeZone,
				likeFields:'host_name,serial_number',
				searchText: '',
			},
            selectionData: [],
            optType: '',
            detailSelectedData: []
		}
	},
	watch:{},
	methods:{ 
		
		init(type, row){
			var vm = this;	
            vm.optType = type;
            if(type == 'view'){
                vm.ruleForm.traceId = row.trace_id;
                vm.getTaskDetail();
            }
            vm.$nextTick(()=>{
                initForm(vm.$refs.ruleForm);
            })
		},
        getTaskDetail(){
            var vm = this,
                urls = '${ctx}/bsc/signaling/querySignalingTraceProperties.action',
                params = {
                    traceId: vm.ruleForm.traceId
                };
            axios.post(urls,stringify(params)).then(function(response){
                var data = response.data;
                vm.ruleForm.enable = data.enable ? data.enable : '1';
                vm.ruleForm.imsi = data.imsi ? data.imsi : '';
                vm.ruleForm.NEInterface = data.ne_interface ? data.ne_interface.split(',') : [];
                vm.detailSelectedData = data.snStr ? data.snStr.split(',') : [];
            })
        },
        // 设备表格查询
        queryDevice(val) {
			this.deviceParams.searchText = val;
		},
        /**
		 * 多选响应
		 * @param selection:选择的数据
		*/
	    batchSelect(selection){
	    	var vm = this,
                deviceList = [];

	    	vm.selectionData = selection;
            if(vm.selectionData.length != 0){
				vm.selectionData.map(function(item){
					deviceList.push(item.serial_number)
				})
			}
			vm.ruleForm.deviceCodes = deviceList.join(',');
            vm.$refs.ruleForm.validateField('deviceCodes');
		},
		// 确定提交 
		submit(){ 
	    	var vm = this, 
                url = '${ctx}/bsc/signaling/addSignalingTraceTask.action',
				params = {
                    enable: vm.ruleForm.enable,
					traceId: vm.ruleForm.traceId,
					timeZone: timeZone,
					imsi: vm.ruleForm.imsi,
					NEInterface: vm.ruleForm.NEInterface.join(','),
                    traceDevice: vm.ruleForm.deviceCodes
				};
	    	vm.$refs.ruleForm.validate((valid) => {
	    		if(valid){
                    axios.post(url,stringify(params)).then(function(response){
                        var data = response.data;
                        if(data["success"]){
                            vm.$message({
                                message:'<%=rb.getString("ChengGong")%>',
                                type:'success'
                            })
                            eventBus.$emit('hander-cancel');
                        }else{
                            vm.$message.error(data["message"]);
                        }
                    })
	    		}
	    	})
		},
		// 关闭弹窗
		closeSlider(){ 
			var vm = this;
			var confirmStr = '<%=rb.getString("QueDingLiKaiDangQianYeMian")%>'
			if(isFormChanged(vm.$refs.ruleForm)){
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					eventBus.$emit('hander-cancel');
				}).catch(() => {})
			}else{
				eventBus.$emit('hander-cancel');
			}
		},
	},
	
	mounted(){
		eventBus.$off('init-config').$on('init-config', this.init); 
		eventBus.$off('hander-ok').$on('hander-ok',this.submit);
		eventBus.$off('close-slide').$on('close-slide',this.closeSlider);
	}
})
</script>
