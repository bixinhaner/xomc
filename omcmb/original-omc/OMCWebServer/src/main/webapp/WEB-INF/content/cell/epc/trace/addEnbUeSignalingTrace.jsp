<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
#addEnbUeTracePage{
    padding: 10px 0px;
    box-sizing: border-box;
}
#addEnbUeTracePage .alarmBottomLine{
	background-color:#E9E9E9;
	width: 100%;
	height: 1px;
	margin-bottom: 30px; 
}
#addEnbUeTracePage .titleStyML{
	margin-left: 40px;
    margin-bottom: 15px;
}
#addEnbUeTracePage .el-form-item{
	margin: 0px 0px 10px 65px;
}
#addEnbUeTracePage .el-form-item .el-input{
    padding-top: 5px;
}
#addEnbUeTracePage .el-checkbox.is-bordered.el-checkbox--small{
	padding: 8px 15px 5px 10px;
}
#addEnbUeTracePage .validate-item .el-input-group__append{
    border:none;
    background:none;
}
#addEnbUeTracePage .validate-item .el-form-item__error{
    display:none;
}
#addEnbUeTracePage .validate-item .el-input__inner{
    width: 330px;
}
#addEnbUeTracePage .is-error .el-input-group__append, #addEnbUeTracePage .is-error .item-tip{
    color:#FA5555;
}
</style>

<div id="addEnbUeTracePage" style="padding: 10px 0px;">
	<el-form :model='ruleForm' :rules="rules" ref="ruleForm" label-position="left" label-width="140px">
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
            <el-input v-model="ruleForm.traceId" size="mini" style='width: 330px;'>
                <template slot="append"><%=rb.getString("FanWei")%>：0~18446744073709551615,Integer</template>
            </el-input>
        </el-form-item>
        <el-form-item label='<%=rb.getString("JieKouLeiXing") %>' prop="NEInterface" style="margin-bottom: 30px;">
            <el-checkbox-group v-model="ruleForm.NEInterface" size="small">
                <el-checkbox label="0" border>S1</el-checkbox>
                <el-checkbox label="1" border style='margin-left: 20px;'>Uu</el-checkbox>
                <!-- <el-checkbox label="2" border style='margin-left: 20px;'>Xn</el-checkbox>
                <el-checkbox label="3" border style='margin-left: 20px;'>X2</el-checkbox>
                <el-checkbox label="4" border style='margin-left: 20px;'>F1-C</el-checkbox> -->
            </el-checkbox-group>
        </el-form-item>
		<div class="alarmBottomLine"></div>
		<div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text commonColor"><%=rb.getString("ZhiXingFangShi")%></span>
		</div>
		<el-form-item label='<%=rb.getString("ChiXuShiChang")%>' prop='duration' class='validate-item'>
            <el-input maxlength=50  v-model="ruleForm.duration" size="mini">
                <template slot="append"><%=rb.getString("ZhengXing")%>[1:1440],<%=rb.getString("DanWei")%>:<%=rb.getString("FenZhongDaXie")%></template>
            </el-input>
        </el-form-item>
	</el-form>
	
</div>

<script type="text/javascript">
var addEnbUeTraceVue = new Vue({
	el:'#addEnbUeTracePage',
	data(){
		var vm = this,
        validateImsi = (rule,value,callback) => {
			var reg = /^[0-9]\d{14,15}$/
			if(value === '' || value === null){
				callback(new Error('Length: 15~16'))
			}else if(reg.test(value)){
				callback();
			}else{
				callback(new Error('Length: 15~16'))
			}
		},
        validateTraceId = (rule,value,callback) => {
            var min = '0',
                max = '18446744073709551615';
			    reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/;
			if(value === '' || value === null){
				callback(new Error('<%=rb.getString("FanWei")%>：0~18446744073709551615,Integer'))
			}else if(reg.test(value) && vm.isLessThan(value,max) && vm.isLessThan(min,value)){
				callback();
			}else{
				callback(new Error('<%=rb.getString("FanWei")%>：0~18446744073709551615,Integer'))
			}
		},
        validateNEInterface = (rule,value,callback) => {
			if(vm.ruleForm.NEInterface.length == 0) {
            	callback('<%=rb.getString("BiTian") %>');
            }else{
            	callback();
            }
		},
        validateDuration = (rule,value,callback) => {
			var reg = /^[0-9]+$/;
			if(value === '' || value === null){
				callback(new Error('error'))
			}else if(reg.test(value) && (value >= 0) && (value <= 1440)){
				callback();
			}else{
				callback(new Error('error'))
			}
		};
		
		return {
			ruleForm:{
                imsi:'',
				traceId: '',
				NEInterface: [],
				duration:'',
			},
			rules:{
                imsi: [{validator: validateImsi,trigger:'blur'}],
                traceId: [{validator: validateTraceId,trigger:'blur'}],
                NEInterface: [{validator: validateNEInterface,trigger:'change'}],
				duration: [{validator: validateDuration,trigger:'blur'}]
			},
		}
	},
	watch:{},
	methods:{ 
		
		init(type, row){
			var vm = this;	
			vm.$nextTick(()=>{
                initForm(vm.$refs.ruleForm);
            })
		},
		// 确定提交 
		submit(){ 
	    	var vm = this, 
                url = '${ctx}/enbue/signaling/addSignalingTraceTask.action', message = '',
				params = {
					traceId: vm.ruleForm.traceId,
					timeZone: timeZone,
					imsi: vm.ruleForm.imsi,
					NEInterface: vm.ruleForm.NEInterface.join(','),
					duration: vm.ruleForm.duration
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
				}).catch(() => {
					
				})
			}else{
				eventBus.$emit('hander-cancel');
			}
		},
        // 较大数字的比较
        isLessThan(val,target){
            var max = target + '',
                num = val + '',
                bool = true,
                list = [];
            if(num.length > max.length || isNaN(num)){
                bool = false;
            }
            if(num.length == max.length){
                for(var i = 0;i<max.length;i++){
                    var isBig = max[i] - num[i] >= 0;
                    list.push(max[i] - num[i] > 0);
                    if(!isBig){
                        var some = list.filter(function(item){return item == true;});
                        if(some.length == 0){
                            bool = false;
                            break;
                        }

                    }
                }
            }
            return bool;
        },
	},
	
	mounted(){
		eventBus.$off('init-config').$on('init-config', this.init); 
		eventBus.$off('hander-ok').$on('hander-ok',this.submit);
		eventBus.$off('close-slide').$on('close-slide',this.closeSlider);
	}
})
</script>
